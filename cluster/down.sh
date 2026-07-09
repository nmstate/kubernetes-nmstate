#!/bin/bash

set -ex

if [[ "$(uname -s)" == "Darwin" ]]; then
    echo "ERROR: make cluster-down is not supported on macOS." >&2
    echo "" >&2
    echo "kubevirtci requires x86_64 Linux. Manage your external cluster directly." >&2
    exit 1
fi

source ./cluster/kubevirtci.sh
kubevirtci::install

./cluster/lldpd-switch.sh down

$(kubevirtci::path)/cluster-up/down.sh
