#!/bin/bash -xe

# This script should be able to execute functional tests against Kubernetes
# cluster on any environment with basic dependencies listed in
# check-patch.packages installed and docker running.
#
# yum -y install automation/check-patch.packages
# automation/check-patch.e2e-k8s.sh

teardown() {
    make cluster-down
    # Don't fail if there is no logs
    cp ${E2E_LOGS}/operator/*.log ${ARTIFACTS} || true
}

main() {
    export KUBEVIRT_PROVIDER='k8s-1.17'
    export KUBEVIRT_NUM_NODES=3 # 1 master, 2 workers
    source automation/check-patch.setup.sh
    cd ${TMP_PROJECT_PATH}

    # Let's fail fast if generated files differ
    make check-gen

    # Let's fail fast if it's not compiling
    make operator

    make cluster-down
    make cluster-up
    trap teardown EXIT SIGINT SIGTERM SIGSTOP
    make E2E_TEST_TIMEOUT=1h E2E_TEST_ARGS="-ginkgo.noColor --junit-output=$ARTIFACTS/junit.functest.xml" test-e2e-operator
}

[[ "${BASH_SOURCE[0]}" == "$0" ]] && main "$@"
