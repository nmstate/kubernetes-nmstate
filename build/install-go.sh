#!/bin/bash -xe

version=$1
arch=""
os=linux
go_mod_version=$1

dnf install -y jq
full_version=$(curl -s https://go.dev/dl/?mode=json |jq -r .[].version |grep "$go_mod_version")

case $(uname -m) in
    x86_64)  arch="amd64";;
    aarch64) arch="arm64" ;;
esac

if [ "$arch" == "" ]; then
    echo "Unknown architecture $(uname -m)"
    exit 1
fi

tarball_url="https://go.dev/dl/${full_version}.${os}-${arch}.tar.gz"

curl --retry 10 -L $tarball_url | tar -C /usr/local -zxf -
