#!/bin/bash -e

organization=kubevirt
commit="9b8707c02d59ee1a7924103b6beca9b9cd010633"

script_dir=$(dirname "$(readlink -f "$0")")
kubevirtci_dir=kubevirtci

rm -rf $kubevirtci_dir
git clone https://github.com/$organization/kubevirtci $kubevirtci_dir
pushd $kubevirtci_dir
git checkout $commit
popd
