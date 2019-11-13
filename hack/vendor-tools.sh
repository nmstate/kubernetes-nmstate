#!/bin/bash -xe
export GOFLAGS=-mod=vendor
export GO111MODULE=on

export GOBIN=$1
tools_file=$2

tools=$(grep "_" $tools_file |  sed 's/.*_ *"//' | sed 's/"//g')
go mod tidy
go mod vendor
for tool in $tools; do
    go install $tool
done
go mod vendor
