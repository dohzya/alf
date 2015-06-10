#!/bin/bash

VERBOSE=${VERBOSE:-false}
KEEP_DIR=${KEEP_DIR:-false}

while getopts vko:a: opt; do
  case "$opt" in
    v) VERBOSE=true;;
    k) KEEP_DIR=true;;
    o) GOOS="$OPTARG";;
    a) GOARCH="$OPTARG";;
    *) echo "bad param (-v -k -o OS -a ARCH)" >&2; exit 1;;
  esac
done
shift $((OPTIND - 1))

export GOOS=${GOOS:-$(go env GOOS)}
export GOARCH=${GOARCH:-$(go env GOARCH)}

FILE_NAME="alf-$GOOS-$GOARCH"

FILES="$FILES go/bin/alf main.js index.html main.css about.html"

$VERBOSE && (
  set -x
  GOOS="$GOOS"
  GOARCH="$GOARCH"
  FILE_NAME="$FILE_NAME"
  FILES="$FILES"
)

rm -rf "$FILE_NAME" &>/dev/null || true
(
  set -e
  if $VERBOSE; then set -x ; fi
  VERBOSE=$VERBOSE ./build.sh
  mkdir "$FILE_NAME"
  cp $FILES "$FILE_NAME/"
  cp config-sample.toml "$FILE_NAME/config.toml"
  cp alf-sample.json "$FILE_NAME/alf.json"
) || exit $?

if [[ $(uname) = "Darwin" ]]; then
  TAR_OPTS="-czf"
else
  TAR_OPTS="czf"
fi
(
  set -e
  if $VERBOSE; then set -x ; fi
  tar $TAR_OPTS "$FILE_NAME.tar.gz" "$FILE_NAME"
) || exit $?
$KEEP_DIR || { rm -rf "$FILE_NAME" || exit $?; }
