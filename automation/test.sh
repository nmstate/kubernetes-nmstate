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
make E2E_TEST_EXTRA_ARGS="-ginkgo.noColor" test/cluster/e2e
