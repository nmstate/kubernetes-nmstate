#!/bin/bash -e

organization=kubevirt
commit="eb9addac961a83dde1ef4f80db131cab58dda1fc"

script_dir=$(dirname "$(readlink -f "$0")")
kubevirtci_dir=kubevirtci

rm -rf $kubevirtci_dir
git clone https://github.com/$organization/kubevirtci $kubevirtci_dir
pushd $kubevirtci_dir
git checkout $commit
popd
