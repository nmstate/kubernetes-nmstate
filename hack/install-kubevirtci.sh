#!/bin/bash -e

organization=kubevirt
commit="767e32264ec8b8ce728fa9b5cd51a5dda68b0bf3"

script_dir=$(dirname "$(readlink -f "$0")")
kubevirtci_dir=kubevirtci

rm -rf $kubevirtci_dir
git clone https://github.com/$organization/kubevirtci $kubevirtci_dir
pushd $kubevirtci_dir
git checkout $commit
popd


