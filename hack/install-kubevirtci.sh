#!/bin/bash -e

organization=kubevirt
commit="effb245dc018e7f502119c1e9f30e703ae8bbe14"

script_dir=$(dirname "$(readlink -f "$0")")
kubevirtci_dir=kubevirtci

rm -rf $kubevirtci_dir
git clone https://github.com/$organization/kubevirtci $kubevirtci_dir
pushd $kubevirtci_dir
git checkout $commit
popd
