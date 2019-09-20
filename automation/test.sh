#!/bin/bash -ex

kubectl() { cluster/kubectl.sh "$@"; }

teardown() {
    kubectl get --all-namespaces event || true
    kubectl get --all-namespaces pod || true
    make cluster-down
}

export KUBEVIRT_PROVIDER=$TARGET

# Make sure that the VM is properly shut down on exit
trap teardown EXIT SIGINT SIGTERM SIGSTOP

make cluster-down
make cluster-up
make cluster-sync
test_args="-ginkgo.v -ginkgo.noColor -test.timeout 20m"
skip_tests=""

# FIXME: Delete it when we migrate to okd4 provider, since os-3.11.0 is not
#        working alright I we don't want to debug not supported providers.
if [[ $KUBEVIRT_PROVIDER =~ os- ]]; then
    skip_tests="move.*default.*IP"
fi

make E2E_TEST_EXTRA_ARGS="$test_args" E2E_TEST_SKIP="$skip_tests" test/e2e
