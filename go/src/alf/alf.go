package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/spf13/viper"
)

/* HELPERS */

func dieIfErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func getIntOrDie(k string) int {
	v := viper.GetInt(k)
	if v == 0 {
		log.Fatalf("Missing conf: %s\n", k)
	}
	return v
}
func getStringOrDie(k string) string {
	v := viper.GetString(k)
	if len(v) == 0 {
		log.Fatalf("Missing conf: %s\n", k)
	}
	return v
}

/* EMAIL */

// SMTPConf is the SMTP conf
type SMTPConf struct {
	enabled  bool
	username string
	password string
	server   string
	port     int
	sender   string
	dest     string
}

func newSMTPConf() SMTPConf {
	if !viper.GetBool("mail_enabled") {
		return SMTPConf{
			enabled: false,
		}
	}
	return SMTPConf{
		enabled:  true,
		username: getStringOrDie("mail_username"),
		password: viper.GetString("mail_password"),
		server:   getStringOrDie("mail_server"),
		port:     getIntOrDie("mail_port"),
		sender:   getStringOrDie("mail_sender"),
		dest:     getStringOrDie("mail_dest"),
	}
}

func (s SMTPConf) host() string {
	return fmt.Sprintf("%s:%d", s.server, s.port)
}

func sendEmail(smtpConf SMTPConf, msg []byte) error {
	if !smtpConf.enabled {
		return nil
	}
	// Set up authentication information.
	auth := smtp.PlainAuth(
		"",
		smtpConf.username,
		smtpConf.password,
		smtpConf.server,
	)
	// Connect to the server, authenticate, set the sender and recipient,
	// and send the email all in one step.
	return smtp.SendMail(
		smtpConf.host(),
		auth,
		smtpConf.sender,
		[]string{smtpConf.dest},
		msg,
	)
}

/* SAVE FILE */

type saveInfoCmd struct {
	infos interface{}
	res   chan error
}

func infoSaverSave(cmd saveInfoCmd, conf Conf) error {
	var err error
	log.Debugf("Received infos: %+v", cmd.infos)
	js, err := json.Marshal(cmd.infos)
	if err != nil {
		return err
	}
	log.Infof("Save infos: %s", js)
	output, errFile := conf.outputFile()
	if errFile == nil {
		fmt.Fprintf(output, "%s\n", js)
		errFile = output.Close()
	}
	err = sendEmail(conf.smtp, []byte("This is the email body."))
	if errFile != nil {
		return errFile
	}
	if err != nil {
		return err
	}
	return nil
}

func infoSaver(in chan saveInfoCmd, stop chan struct{}, conf Conf) {
	for {
		select {
		case cmd := <-in:
			cmd.res <- infoSaverSave(cmd, conf)
		case <-stop:
			return
		}
	}
}

func saveInfos(c chan saveInfoCmd, infos interface{}) error {
	res := make(chan error)
	c <- saveInfoCmd{infos: infos, res: res}
	return <-res
}

/* MAIN */

// Conf is the main conf
type Conf struct {
	smtp       SMTPConf
	outputPath string
}

func (c *Conf) outputFile() (*os.File, error) {
	return os.OpenFile(c.outputPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0777)
}

func newConf() Conf {
	return Conf{
		smtp:       newSMTPConf(),
		outputPath: viper.GetString("output_file"),
	}
}

// Infos contains the informations to save
type Infos struct {
	At          time.Time `json:"at"`
	Date        string    `json:"date"`
	Start       string    `json:"start"`
	Finish      string    `json:"finish"`
	Name        string    `json:"name"`
	Contact     string    `json:"contact"`
	ContactType string    `json:"contactType"`
}

func (i Infos) validate() error {
	var empty Infos
	if i.Date == empty.Date {
		return fmt.Errorf("missing date")
	}
	if i.Start == empty.Start {
		return fmt.Errorf("missing start")
	}
	if i.Finish == empty.Finish {
		return fmt.Errorf("missing finish")
	}
	if i.Name == empty.Name {
		return fmt.Errorf("missing name")
	}
	if i.Contact == empty.Contact {
		return fmt.Errorf("missing contact")
	}
	if i.ContactType == empty.ContactType {
		return fmt.Errorf("missing contactType")
	}
	return nil
}

// JSON allows to easily create JSON structures
type JSON map[string]interface{}

func jsonRespond(w http.ResponseWriter, content JSON, code int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	msg, err := json.Marshal(content)
	if err != nil {
		log.Errorf("Error while formatting response msg: %v\n", err)
		code = 500
		msg = []byte(`{"error":"Error occured during response"}`)
	}
	w.WriteHeader(code)
	fmt.Fprintf(w, "%s\n", msg)
}
func jsonError(msg string) JSON {
	return JSON{"error": msg}
}
func jsonErrorf(format string, args ...interface{}) JSON {
	return JSON{"error": fmt.Sprintf(format, args...)}
}

func requestHandler(c chan saveInfoCmd) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		// CORS
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Add("Access-Control-Allow-Methods", "POST")
		w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.Header().Set("Allow", "POST,OPTIONS")
			return
		}

		if r.Method != "POST" {
			http.NotFound(w, r)
			return
		}

		var err error
		var infos Infos

		err = json.NewDecoder(r.Body).Decode(&infos)
		if err != nil {
			log.Debugf("Invalid JSON: %v", err)
			jsonRespond(w, jsonError("Invalid content: invalid JSON"), 400)
			return
		}
		infos.At = time.Now()
		err = infos.validate()
		if err != nil {
			log.Debugf("Invalid content: %v", err)
			jsonRespond(w, jsonErrorf("Invalid content: %v", err), 400)
			return
		}
		err = saveInfos(c, infos)
		if err != nil {
			log.Errorf("Error occured while saving: %v", err)
			jsonRespond(w, jsonError("Error occured"), 500)
			return
		}

		jsonRespond(w, JSON{"saved": true}, 200)
	}
}

func main() {
	viper.SetConfigName("config")
	dieIfErr(viper.ReadInConfig())

	viper.SetDefault("log_level", "info")
	viper.SetDefault("http_port", 8080)
	viper.SetDefault("output_file", "output.json")
	viper.SetDefault("mail_enabled", false)

	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)

	conf := newConf()

	logLevel, err := log.ParseLevel(viper.GetString("log_level"))
	dieIfErr(err)
	log.SetLevel(logLevel)

	log.Debug("Starting info saver")
	c := make(chan saveInfoCmd)
	stop := make(chan struct{})
	go infoSaver(c, stop, conf)

	port := viper.GetInt("http_port")
	http.HandleFunc("/infos", requestHandler(c))
	log.Infof("Starting the HTTP server on port %d...", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))

	log.Infof("Stopping...")
	stop <- struct{}{}

	log.Info("Bye")
}
