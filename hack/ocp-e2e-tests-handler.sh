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
export PRIMARY_NIC=enp2s0
export FIRST_SECONDARY_NIC=enp3s0
export SECOND_SECONDARY_NIC=enp4s0
export FLAKE_ATTEMPTS="${FLAKE_ATTEMPTS:-3}"

SKIPPED_TESTS="user-guide|bridged|\
when desiredState is updated with ovs-bridge with linux bond as port" # https://bugzilla.redhat.com/show_bug.cgi?id=2005240 is not yet fixed in nmstate 1.2

if [ "${CI}" == "true" ]; then
    source ${SHARED_DIR}/fix-uid.sh
    export SSH=./hack/ssh-ci.sh
else
    export SSH=./hack/ssh.sh
fi

if oc get ns openshift-ovn-kubernetes &> /dev/null; then
    # We are using OVNKubernetes -> use enp1s0 as primary nic
    export PRIMARY_NIC=enp1s0
    SKIPPED_TESTS+="|NodeNetworkConfigurationPolicy bonding default interface|\
with ping fail|\
when connectivity to default gw is lost after state configuration|\
when name servers are lost after state configuration|\
when name servers are wrong after state configuration"
fi

make cluster-sync-operator
oc create -f test/e2e/nmstate.yaml
# On first deployment, it can take a while for all of the pods to come up
# First wait for the handler pods to be created
while ! oc get pods -n nmstate | grep handler; do sleep 1; done
# Then wait for them to be ready
while oc get pods -n nmstate | grep "0/1"; do sleep 1; done
# NOTE(bnemec): The test being filtered with "bridged" was re-enabled in 4.8, but seems to be consistently failing on OCP.
make test-e2e-handler E2E_TEST_ARGS="--skip=\"${SKIPPED_TESTS}\" --flake-attempts=${FLAKE_ATTEMPTS}" E2E_TEST_TIMEOUT=4h
