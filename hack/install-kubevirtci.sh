#!/bin/bash -e

organization=kubevirtci
commit="db3913a3a919bbc20c3b7044dffc70c009fdc3bd"

script_dir=$(dirname "$(readlink -f "$0")")
kubevirtci_dir=kubevirtci

rm -rf $kubevirtci_dir
git clone https://github.com/$organization/kubevirtci $kubevirtci_dir
pushd $kubevirtci_dir
git checkout $commit
popd
