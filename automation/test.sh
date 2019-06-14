#!/bin/bash -ex

kubectl() { cluster/kubectl.sh "$@"; }

export KUBEVIRT_PROVIDER=$TARGET

# Make sure that the VM is properly shut down on exit
trap '{ make cluster-down; }' EXIT SIGINT SIGTERM SIGSTOP

make cluster-down
make cluster-up
make cluster-sync test/cluster/e2e
