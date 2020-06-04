#!/bin/bash

set -e

current_branch=$(git branch --show-current)
versions=($(git tag --sort version:refname --merged qinqon/$current_branch |grep "^v[0-9]*\.[0-9]*\.[0-9]*$"))
echo ${versions[$1]}
