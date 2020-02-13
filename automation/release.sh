#!/bin/bash -xe

# This script tags current version and create a new release at github and pushes
# to quay.io
# To run it just do proper docker login and pass the git hub user password/token as
# GITHUB_USER and GITHUB_TOKEN env variables.

# To pass user/password from automations, idea is taken from [1]
# [1] https://stackoverflow.com/questions/8536732/can-i-hold-git-credentials-in-environment-variables
git config credential.helper '!f() { sleep 1; echo "username=${GITHUB_USER}"; echo "password=${GITHUB_TOKEN}"; }; f'

git tag foobar

git push https://github.com/qinqon/kubernetes-nmstate foobar

exit 1

source automation/check-patch.setup.sh
cd ${TMP_PROJECT_PATH}
make \
    IMAGE_REGISTRY=${IMAGE_REGISTRY:-quay.io}  \
    IMAGE_REPO=${IMAGE_REPO:-nmstate} \
    release
