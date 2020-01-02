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
    export KUBEVIRT_PROVIDER='k8s-1.15.1'
    export KUBEVIRT_NUM_NODES=2
    source automation/check-patch.e2e.setup.sh
    cd ${TMP_PROJECT_PATH}

    make cluster-down
    make cluster-up
    trap teardown EXIT SIGINT SIGTERM SIGSTOP
    make cluster-sync
    ip a
    ip r
    make \
        E2E_TEST_ARGS='-ginkgo.noColor -ginkgo.skip .*NNS.*cleanup.*' \
        test/e2e
    make \
        E2E_TEST_ARGS='-ginkgo.noColor -ginkgo.focus .*NNS.*cleanup.*' \
        test/e2e
}

[[ "${BASH_SOURCE[0]}" == "$0" ]] && main "$@"
