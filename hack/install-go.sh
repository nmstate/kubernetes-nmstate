#!/bin/bash -xe

destination=$1
version=$(grep "^go " go.mod |awk '{print $2}')
tarball=go$version.linux-amd64.tar.gz
url=https://dl.google.com/go/

mkdir -p $destination
curl -L $url/$tarball -o $destination/$tarball
tar -xf $destination/$tarball -C $destination
