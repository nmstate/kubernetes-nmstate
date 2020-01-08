#!/bin/bash -xe

# This script publish kubernetes-nmstate-handler by default at quay.io/nmstate
# organization to publish elsewhere export the following env vars
# IMAGE_REGISTRY
# IMAGE_REPO
# To run it just do proper docker login and automation/publish.sh

source automation/check-patch.setup.sh
cd ${TMP_PROJECT_PATH}
make \
    IMAGE_REGISTRY=${IMAGE_REGISTRY:-quay.io}  \
    IMAGE_REPO=${IMAGE_REPO:-nmstate} \
    push-handler
