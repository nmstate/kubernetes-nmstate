#!/bin/bash

if [ -n "$(gofmt -l cmd/ pkg/)" ]; then
    echo 'Go code is not formatted:'
    gofmt -d cmd/ pkg/
    exit 1
fi
