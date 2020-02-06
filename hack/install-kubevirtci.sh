#!/bin/bash -e

organization=kubevirt
commit="0c3911794ad2b79a61a4bc7462c236251a73866f"

script_dir=$(dirname "$(readlink -f "$0")")
kubevirtci_dir=kubevirtci

rm -rf $kubevirtci_dir
git clone https://github.com/$organization/kubevirtci $kubevirtci_dir
pushd $kubevirtci_dir
git checkout $commit
popd
