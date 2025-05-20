#!/bin/bash -xe

architecture=""
case $(uname -m) in
    x86_64) architecture="amd64" ;;
    arm64)    architecture="arm64" ;;
esac

curl -L https://github.com/operator-framework/operator-registry/releases/download/v1.24.0/"$(uname -s | tr "[:upper:]" "[:lower:]")"-${architecture}-opm -o /tmp/opm
chmod 755 /tmp/opm
/tmp/opm $@
