#!/bin/bash -e

organization=kubevirt
commit="95096c8189c8b620ddc1310e12388df2190e1cc8"

script_dir=$(dirname "$(readlink -f "$0")")
kubevirtci_dir=kubevirtci

rm -rf $kubevirtci_dir
git clone https://github.com/$organization/kubevirtci $kubevirtci_dir
pushd $kubevirtci_dir
git checkout $commit
popd
