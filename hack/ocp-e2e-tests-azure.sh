#!/bin/bash

set -ex

export KUBEVIRT_PROVIDER=external
export IMAGE_BUILDER="${IMAGE_BUILDER:-podman}"
export DEV_IMAGE_REGISTRY="${DEV_IMAGE_REGISTRY:-quay.io}"
export KUBEVIRTCI_RUNTIME="${KUBEVIRTCI_RUNTIME:-podman}"
export FLAKE_ATTEMPTS="${FLAKE_ATTEMPTS:-3}"
export NAMESPACE="${HANDLER_NAMESPACE:-nmstate}"

make cluster-sync-operator
oc create -f test/e2e/nmstate.yaml
# On first deployment, it can take a while for all of the pods to come up
# First wait for the handler pods to be created
while ! oc get pods -n ${NAMESPACE} | grep handler; do sleep 1; done
# Then wait for them to be ready
while oc get pods -n ${NAMESPACE} | grep "0/1"; do sleep 1; done

# This is dedicated runner for Azure. The only NIC is eth0 and we use `oc debug node`
# instead of SSH to access the nodes.
FOCUS_TESTS="Dns configuration"
SKIPPED_TESTS="with DHCP"
export PRIMARY_NIC=eth0
export ENV_WITH_ONLY_ONE_NIC=True
export SSH="./hack/ssh-via-kubectl.sh"
make test-e2e-handler E2E_TEST_ARGS="--focus=\"${FOCUS_TESTS}\" --skip=\"${SKIPPED_TESTS}\" --flake-attempts=${FLAKE_ATTEMPTS}" E2E_TEST_TIMEOUT=4h
