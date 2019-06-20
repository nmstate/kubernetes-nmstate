#!/bin/bash -e

# TODO: This is temporal while [1] is merged
#       [1] https://github.com/kubevirt/kubevirtci/pull/108
organization=qinqon
commit="9f21fcda8643d979620edd4c658869220fd36822"

script_dir=$(dirname "$(readlink -f "$0")")
kubevirtci_dir=kubevirtci

rm -rf $kubevirtci_dir
git clone https://github.com/$organization/kubevirtci $kubevirtci_dir
pushd $kubevirtci_dir
git checkout $commit
popd


