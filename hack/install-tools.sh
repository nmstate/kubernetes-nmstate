#!/bin/bash

set -xe

tools_file=tools.go
tools=$(grep "_" $tools_file |  sed 's/.*_ *"//' | sed 's/"//g')

for tool in $tools; do
    $GO install $tool
done
