#!/bin/bash -e

organization=kubevirt
commit="4cacc40e97b7a22f354250e6e50630f1e30fd6fb"

script_dir=$(dirname "$(readlink -f "$0")")
kubevirtci_dir=kubevirtci

rm -rf $kubevirtci_dir
git clone https://github.com/$organization/kubevirtci $kubevirtci_dir
pushd $kubevirtci_dir
git checkout $commit
popd
