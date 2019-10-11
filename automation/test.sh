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
test_args="-ginkgo.noColor"
focus_tests=""
skip_tests=""

if [[ $SCRIPT_NAME =~ default-bridge ]]; then
    focus_tests=".*move.*default.*IP.*"
else
    skip_tests=".*move.*default.*IP.*"
fi

if [[ $SCRIPT_NAME =~ node-removal ]]; then
    focus_test=".*NNS.*cleanup.*"
fi

make E2E_TEST_EXTRA_ARGS="$test_args" E2E_TEST_FOCUS="$focus_tests" E2E_TEST_SKIP="$skip_tests" test/e2e
