#!/bin/bash

# Make sure to set the IMAGE_REPO env variable to your quay.io username
# before running this script.

# Additionally, the e2e tests rely on extra nics being configured on the
# node. If running from dev-scripts, it will be necessary to configure it to
# deploy the extra nics.
# See https://github.com/openshift-metal3/dev-scripts/pull/1286 for an example.

set -ex

export KUBEVIRT_PROVIDER=external
export IMAGE_BUILDER="${IMAGE_BUILDER:-podman}"
export DEV_IMAGE_REGISTRY="${DEV_IMAGE_REGISTRY:-quay.io}"
export KUBEVIRTCI_RUNTIME="${KUBEVIRTCI_RUNTIME:-podman}"
export FLAKE_ATTEMPTS="${FLAKE_ATTEMPTS:-3}"

if [ "${CI}" == "true" ]; then
    source ${SHARED_DIR}/fix-uid.sh
    export SSH=./hack/ssh-ci.sh
else
    export SSH=./hack/ssh.sh
fi

make cluster-sync-operator
make test-e2e-operator E2E_TEST_ARGS="--flake-attempts=${FLAKE_ATTEMPTS}"
