#!/bin/bash -ex

kubectl() { cluster/kubectl.sh "$@"; }

export CLUSTER_PROVIDER=$TARGET

# Make sure that the VM is properly shut down on exit
trap '{ make cluster-down; }' EXIT SIGINT SIGTERM SIGSTOP

make cluster-down
make cluster-up
make cluster-sync
make functest
