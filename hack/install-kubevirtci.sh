#!/bin/bash -e

organization=kubevirt
commit="1c2922cd9ebaabf7006dc68aa8df70ad84d96d4b"

script_dir=$(dirname "$(readlink -f "$0")")
kubevirtci_dir=kubevirtci

rm -rf $kubevirtci_dir
git clone https://github.com/$organization/kubevirtci $kubevirtci_dir
pushd $kubevirtci_dir
git checkout $commit
popd
