#!/usr/bin/env bash

set -x -o errexit -o pipefail

name=$1
destdir=$2

export GOOS=linux
export CGO_ENABLED=0

if [ "$ARCHS" == "" ]; then
    go build -o $destdir/$name.manager.$GOOS-$(go env GOARCH) ./cmd/$name
else
    for arch in $ARCHS; do
	    GOARCH=$arch go build -o $destdir/$name.manager.$GOOS-$arch ./cmd/$name
    done
fi
