#!/bin/bash -e

organization=kubevirt
commit="6132f2efcf2aa2d48f256a409fe90cc0b74c3563"

script_dir=$(dirname "$(readlink -f "$0")")
kubevirtci_dir=kubevirtci

rm -rf $kubevirtci_dir
git clone https://github.com/$organization/kubevirtci $kubevirtci_dir
pushd $kubevirtci_dir
git checkout $commit
popd
