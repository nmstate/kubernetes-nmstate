#!/bin/bash -xe

# This script should be able to execute functional tests against OKD cluster on
# any environment with basic dependencies listed in check-patch.packages
# installed and docker running.
#
# yum -y install automation/check-patch.packages
# automation/check-patch.e2e-okd.sh

teardown() {
    make cluster-down
}

main() {
    export KUBEVIRT_PROVIDER='ocp-4.3'

    source automation/check-patch.setup.sh
    cd ${TMP_PROJECT_PATH}

    # Let's fail fast if it's not compiling
    make handler

    make cluster-down
    make cluster-up
    trap teardown EXIT SIGINT SIGTERM SIGSTOP
    make cluster-sync
    make E2E_TEST_ARG="-ginkgo.noColor" test/e2e
    cp $(find . -name "*junit*.xml") $ARTIFACTS
}

[[ "${BASH_SOURCE[0]}" == "$0" ]] && main "$@"
