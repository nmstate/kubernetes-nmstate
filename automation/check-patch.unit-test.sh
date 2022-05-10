#!/bin/bash -e

# This script should be able to execute functional tests against Kubernetes
# cluster on any environment with basic dependencies listed in
# check-patch.packages installed and docker running.
#
# yum -y install automation/check-patch.packages
# automation/check-patch.e2e-k8s.sh

main() {
    source automation/check-patch.setup.sh
    cd ${TMP_PROJECT_PATH}
    make all
    make UNIT_TEST_ARGS="--output-dir=$ARTIFACTS --no-color --compilers=2" test/unit
}

[[ "${BASH_SOURCE[0]}" == "$0" ]] && main "$@"
