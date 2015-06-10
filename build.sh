#!/bin/bash

VERBOSE=${VERBOSE:-false}

while getopts vko:a: opt; do
  case "$opt" in
    v) VERBOSE=true;;
    o) export GOOS="$OPTARG";;
    a) export GOARCH="$OPTARG";;
    *) echo "bad param (-v -o OS -a ARCH)" >&2; exit 1;;
  esac
done
shift $((OPTIND - 1))

# GO
(
  set -e
  if $VERBOSE; then set -x ; fi
  # env must set GOPATH (GOPATH=/path-to-alf/go)
  . ./env

  # cd $GOPATH/src/github.com/dohzya/alf
  cd $GOPATH/src/alf

  gb build
  # gometalinter -D structcheck || exit $?
) || exit $?

# JS
(
  set -e
  if $VERBOSE; then set -x ; fi
  npm run build
)
