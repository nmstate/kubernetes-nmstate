#!/bin/bash

set -e

git fetch --tags > /dev/null 2>&1|| true
current_branch=$(git branch --show-current)
versions=($(git tag --sort version:refname --merged $current_branch |grep "^v[0-9]*\.[0-9]*\.[0-9]*$"))
echo ${versions[$1]}
