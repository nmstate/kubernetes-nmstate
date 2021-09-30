#!/usr/bin/env bash

set -x -o errexit -o pipefail

destdir=$1

export GOOS=linux

if [ "$ARCHS" == "" ]; then
    go build -o $destdir/manager.$GOOS-$(go env GOARCH) main.go
else
    for arch in $ARCHS; do
	    GOARCH=$arch go build -o $destdir/manager.$GOOS-$arch main.go
    done
fi
