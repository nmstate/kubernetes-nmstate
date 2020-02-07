#!/bin/bash -xe
tools_file=$2
tools=$(grep "_" $tools_file |  sed 's/.*_ *"//' | sed 's/"//g')
for tool in $tools; do
    go install $tool
done
