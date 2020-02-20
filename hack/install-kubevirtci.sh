#!/bin/bash -e

organization=kubevirt
commit="8c311c6ede0400b510aee4eeb37dac5068d92fff"

script_dir=$(dirname "$(readlink -f "$0")")
kubevirtci_dir=kubevirtci

rm -rf $kubevirtci_dir
git clone https://github.com/$organization/kubevirtci $kubevirtci_dir
pushd $kubevirtci_dir
git checkout $commit
popd
