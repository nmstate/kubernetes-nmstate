#!/bin/bash -e

organization=kubevirt
commit="db8c24bf830bb927f01829e6c9f083627fe6b832"

script_dir=$(dirname "$(readlink -f "$0")")
kubevirtci_dir=kubevirtci

rm -rf $kubevirtci_dir
git clone https://github.com/$organization/kubevirtci $kubevirtci_dir
pushd $kubevirtci_dir
git checkout $commit
popd
