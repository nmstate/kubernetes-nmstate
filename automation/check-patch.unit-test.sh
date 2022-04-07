#!/bin/bash -e

# This script should be able to execute functional tests against Kubernetes
# cluster on any environment with basic dependencies listed in
# check-patch.packages installed and docker running.
#
# yum -y install automation/check-patch.packages
# automation/check-patch.e2e-k8s.sh

install_nmstate_devel() {
  dnf install -b -y dnf-plugins-core
  dnf copr enable -y nmstate/nmstate-git
  dnf install -b -y nmstate-devel
}

main() {
    source automation/check-patch.setup.sh
    sudo install_nmstate_devel
    cd ${TMP_PROJECT_PATH}
    make all
    make UNIT_TEST_ARGS="--output-dir=$ARTIFACTS --no-color --compilers=2" test/unit
}

[[ "${BASH_SOURCE[0]}" == "$0" ]] && main "$@"
