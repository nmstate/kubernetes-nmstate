#!/bin/bash

set -euo pipefail

version=${1:?helm version is required}
destination=${2:?helm destination is required}
os=$(go env GOOS)
arch=$(go env GOARCH)
archive=helm-${version}-${os}-${arch}.tar.gz
url=https://get.helm.sh/${archive}
tmpdir=

cleanup() {
    rm -rf "${tmpdir}"
}

mkdir -p "$(dirname "${destination}")"
tmpdir=$(mktemp -d "$(dirname "${destination}")/helm.tmp.XXXXXX")
trap cleanup EXIT

curl -fsSL "${url}" -o "${tmpdir}/helm.tar.gz"
tar -xzOf "${tmpdir}/helm.tar.gz" "${os}-${arch}/helm" > "${tmpdir}/helm"
chmod +x "${tmpdir}/helm"
mv "${tmpdir}/helm" "${destination}"
