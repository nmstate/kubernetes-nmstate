#!/bin/bash -xe

version=$1
arch=$(uname -m)
os=linux

dnf install -y jq

case $arch in
    x86_64)  arch="amd64" ;;
    aarch64) arch="arm64" ;;
esac

echo "Installing Go ${version} for architecture: ${arch}"

tarball_url="https://go.dev/dl/go${version}.${os}-${arch}.tar.gz"

curl --retry 10 -L $tarball_url | tar -C /usr/local -zxf -
