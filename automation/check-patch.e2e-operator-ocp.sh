#!/bin/bash -xe

# This script should be able to execute functional tests against OKD cluster on
# any environment with basic dependencies listed in check-patch.packages
# installed and docker running.
#
# yum -y install automation/check-patch.packages
# automation/check-patch.e2e-okd.sh

teardown() {
    make cluster-down
    cp $(find . -name "*junit*.xml") $ARTIFACTS
    # Don't fail if there is no logs
    cp ${E2E_LOGS}/*.log ${ARTIFACTS} || true
}

main() {
    export KUBEVIRT_PROVIDER='ocp-4.4'

    source automation/check-patch.setup.sh
    cd ${TMP_PROJECT_PATH}

    # Let's fail fast if generated files differ
    make check-gen

    # Let's fail fast if it's not compiling
    make operator

    make cluster-down
    make cluster-up
    trap teardown EXIT SIGINT SIGTERM SIGSTOP
    make cluster-sync-operator
    make E2E_TEST_TIMEOUT=2h E2E_TEST_ARGS="-ginkgo.noColor " test-e2e-operator
}

[[ "${BASH_SOURCE[0]}" == "$0" ]] && main "$@"
