/* @flow */
/* jshint asi:true */

var Q = require("q")
var _ = require("lodash")
var moment = require("moment")

require("moment/locale/ja")
moment.locale("ja")

window.ALF = (function (localStorage) {
  "use strict";

  function getParameterByName(name) {
    name = name.replace(/[\[]/, "\\[").replace(/[\]]/, "\\]")
    var regex = new RegExp("[\\?&]" + name + "=([^&#]*)")
    var results = regex.exec(location.search)
    return results === null ? "" : decodeURIComponent(results[1].replace(/\+/g, " "))
  }

  var debugMode = getParameterByName("debug") === "true"

  var debug
  if (debugMode) {
    debug = console
  } else {
    function noop() {}
    debug = {}
    debug.log = noop
    debug.group = noop
    debug.groupEnd = noop
  }

  var helpers = {
    moment: moment,
    _: _
  }

  var ALF = {}

  function ajax(opts) {
    var deferred = Q.defer()
    var request = new XMLHttpRequest()
    if (typeof opts == "string") {
      opts = {url: opts, method: "GET"}
    }
    request.open(opts.method || "GET", opts.url, true)
    request.onreadystatechange = function(e) {
      if (request.readyState !== 4) {
        return
      }
      if (request.status >= 400) {
        deferred.reject(new Error("Server responded with a status of " + request.status))
      } else {
        try {
          deferred.resolve(JSON.parse(e.target.responseText))
        } catch (err) {
          deferred.reject(err)
        }
      }
    }
    if (typeof opts.data == "object") {
      request.setRequestHeader("Content-type", "application/json")
      request.send(JSON.stringify(opts.data))
    } else {
      request.send(opts.data || void 0)
    }
    return deferred.promise
  }

  function renderTemplate(template, data) {
    return _.template(template)(_.assign({}, data, helpers))
  }

  function findCalledEl(container, el, filter) {
    while (el !== container) {
      if (filter(el)) {
        return el
      }
      el = el.parentElement
    }
  }

  function filterHasAttribute(attr) {
    return function (el) {
      return el.getAttribute(attr)
    }
  }

  function showPopup(el, ctx) {
    debug.group("showPopup")
    debug.log("el:", el)
    debug.log("ctx:", ctx)
    var popup = findPopupByEl(el, ctx)
    debug.log("popup:", popup)
    var storage = {}
    if (localStorage) {
      _.each(popup.storage, function (field) {
        storage[field] = localStorage.getItem(field)
      })
    }
    popup.el.innerHTML = renderTemplate(popup.template.innerHTML, {
      data: popup.data,
      storage: storage
    })
    setNotice(popup, "new")
    ctx.popupBg.style.display = "block"
    popup.el.style.display = "block"
    popup.el.onclick = function (e) {
      debug.group("popup::click")
      debug.log("target:", e.target)
      var el = findCalledEl(popup.el, e.target, filterHasAttribute("data-popup-trigger"))
      debug.log("el:", el)
      if (el) {
        var trigger = el.getAttribute("data-popup-trigger")
        debug.log("trigger:", trigger)
        switch (trigger) {
          case "hide": hidePopup(popup, ctx)
          case "action": popup.action(popup, ctx)
        }

      }
      debug.groupEnd()
    }
    ctx.popupBg.onclick = function () { hidePopup(popup, ctx) }
    debug.groupEnd()
  }

  function hidePopup(popup, ctx) {
    popup.el.style.display = "none"
    ctx.popupBg.style.display = "none"
    ctx.popupBg.onclick = undefined
    _.each(popup.el.querySelectorAll(".popup-button-cancel"), function (btn) {
      btn.onclick = undefined
    })
  }

  function findPopupByEl(el, ctx) {
    var data = {}
    _.each(el.attributes, function (attr) {
      if (/^data-/.test(attr.name)) {
        data[attr.name.slice(5)] = attr.value
      }
    })
    return _.assign({}, ctx.popups[el.getAttribute("data-popup")], {data: data})
  }

  function setNotice(popup, status) {
    debug.group("setNotice")
    debug.log("popup:", popup)
    debug.log("status:", status)
    popup.el.setAttribute("data-status", status)
    debug.groupEnd()
  }

  ALF.applyCourse = function (popup, ctx) {
    var fields = {}
    var errors = false
    _.each(popup.el.querySelectorAll(".popup-field input[name], .popup-field select[name]"), function (field) {
      if (field.value === "") {
        var field = popup.el.querySelector("[name=" + field.name + "]")
        field.className = field.className + " popup-field-error"
        errors = true
        if (localStorage) {
          localStorage.removeItem(field.name)
        }
      } else {
        var field = popup.el.querySelector("[name=" + field.name + "]")
        field.className = field.className.replace(" popup-field-error", "")
        fields[field.name] = field.value
        if (localStorage) {
          localStorage.setItem(field.name, field.value)
        }
      }
    })
    if (errors) {
      setNotice(popup, "invalid")
    } else {
      debug.log("applyCourse :: fields = ", fields)
      ajax({
        url: "http://127.0.0.1:8080/infos",
        method: "POST",
        data: fields
      }).then(
        function (res) { setNotice(popup, "success") },
        function (err) { setNotice(popup, "error") }
      ).done()
    }
  }

  ALF.start = function (ctx) {
    if (debugMode) {
      window.moment = moment
      window._ = _
      window.Q = Q
      Q.longStackSupport = true;
    } // debugMode

    ajax(ctx.dataFile).then(function (data) {
      debug.log("data: ", data)
      ctx.cal.innerHTML = renderTemplate(ctx.template.innerHTML, data)

      ctx.cal.onclick = function (e) {
        debug.group("cal::click")
        debug.log("target:", e.target)
        var el = findCalledEl(ctx.cal, e.target, filterHasAttribute("data-popup"))
        debug.log("el:", el)
        if (el) showPopup(el, ctx)
        debug.groupEnd()
      }

    }).done()
  }

  return ALF

})(window.localStorage);
