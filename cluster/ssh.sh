#!/bin/bash

set -e

source ./cluster/kubevirtci.sh
kubevirtci::install

$(kubevirtci::path)/cluster-up/cli.sh ssh "$@"
