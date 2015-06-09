#!/bin/bash -x

# env must set GOPATH (GOPATH=/path-to-alf/go)
. ./env

# cd $GOPATH/src/github.com/dohzya/alf
cd $GOPATH/src/alf

go install || exit $?
gometalinter -D structcheck || exit $?
