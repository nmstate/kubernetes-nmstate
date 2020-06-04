#!/bin/bash

set -xe

version_type=$1
remote=git@github.com:nmstate/kubernetes-nmstate.git
bumped_version=$(hack/bump-version.sh $version_type)

git tag $bumped_version
git push $remote $bumped_version
