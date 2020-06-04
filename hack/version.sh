#!/bin/bash

set -e

# Return last tag with format vx.y.z
versions=($(git for-each-ref --sort=creatordate --format '%(refname)' refs/tags |grep "refs/tags/v[0-9]*\.[0-9]*\.[0-9]*$"))
echo ${versions[$1]} | sed "s#refs/tags/##g"
