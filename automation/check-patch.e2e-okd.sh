#!/bin/bash -xe

# This script should be able to execute functional tests against OKD cluster on
# any environment with basic dependencies listed in check-patch.packages
# installed and docker running.
#
# yum -y install automation/check-patch.packages
# automation/check-patch.e2e-okd.sh

# FIXME: Delete this when okd lane works
exit 0

teardown() {
    make cluster-down
    cp $(find . -name "*junit*.xml") $ARTIFACTS
    [ -d ${E2E_LOGS} ] && cp ${E2E_LOGS}/*.log ${ARTIFACTS}
}

main() {
    export KUBEVIRT_PROVIDER='okd-4.1'

    source automation/check-patch.setup.sh
    cd ${TMP_PROJECT_PATH}

    # Let's fail fast if it's not compiling
    make handler

    make cluster-down
    make cluster-up
    trap teardown EXIT SIGINT SIGTERM SIGSTOP
    make cluster-sync
    make E2E_TEST_TIMEOUT=1h E2E_TEST_ARG="-ginkgo.noColor -ginkgo.skip .*OVS.* " test/e2e
}

[[ "${BASH_SOURCE[0]}" == "$0" ]] && main "$@"
