#!/bin/bash -xe

architecture=""
case $(uname -m) in
    i386)   architecture="386" ;;
    i686)   architecture="386" ;;
    x86_64) architecture="amd64" ;;
    arm)    architecture="arm64" ;;
esac

destination=$1
version=$(grep "^go " go.mod |awk '{print $2}')
tarball=go$version.linux-${architecture}.tar.gz
url=https://dl.google.com/go/

mkdir -p $destination
curl -L $url/$tarball -o $destination/$tarball
tar -xf $destination/$tarball -C $destination
