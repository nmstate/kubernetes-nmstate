#!/bin/bash -xe

# This script should be able to execute functional tests against Kubernetes
# cluster on any environment with basic dependencies listed in
# check-patch.packages installed and docker running.
#
# yum -y install automation/check-patch.packages
# automation/check-patch.e2e-k8s.sh

teardown() {
    make cluster-down
}

main() {

    export KUBEVIRT_PROVIDER='k8s-1.16.2'
    export KUBEVIRT_NUM_NODES=2
    source automation/check-patch.setup.sh
    cd ${TMP_PROJECT_PATH}

    # Let's fail fast if it's not compiling
    make handler

    make cluster-down
    make cluster-up
    trap teardown EXIT SIGINT SIGTERM SIGSTOP
    make cluster-sync
    make E2E_TEST_ARG="-ginkgo.noColor" test/e2e
}

[[ "${BASH_SOURCE[0]}" == "$0" ]] && main "$@"
