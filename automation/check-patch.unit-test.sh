#!/bin/bash -e

# This script should be able to execute functional tests against Kubernetes
# cluster on any environment with basic dependencies listed in
# check-patch.packages installed and podman / docker running.
#
# yum -y install automation/check-patch.packages
# automation/check-patch.e2e-k8s.sh

main() {
    source automation/check-patch.setup.sh
    cd ${TMP_PROJECT_PATH}
    export ARCHS="amd64 arm64"
    make all
    # Also build the handler with nmstate from the git Copr (NMSTATE_SOURCE=git),
    # the same code path exercised by periodic-knmstate-e2e-handler-k8s-latest.
    # This shifts-left multi-arch Copr/chroot regressions to per-PR CI instead of
    # only catching them in the nightly periodic job.
    make NMSTATE_VERSION=latest handler
    make test-reporter
    make UNIT_TEST_ARGS="--output-dir=$ARTIFACTS --no-color --compilers=2" test/unit
}

[[ "${BASH_SOURCE[0]}" == "$0" ]] && main "$@"
