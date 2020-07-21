#!/bin/bash

set -e

git fetch https://github.com/nmstate/kubernetes-nmstate --tags > /dev/null 2>&1|| true
current_branch=$(git rev-parse --abbrev-ref HEAD)
versions=($(git tag --sort version:refname --merged $current_branch |grep "^v[0-9]*\.[0-9]*\.[0-9]*$"))
echo ${versions[$1]}
