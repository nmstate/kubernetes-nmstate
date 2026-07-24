#!/bin/bash

set -ex

source ./cluster/lima.sh
lima::ensure_linux

source ./cluster/sync-common.sh
source ./cluster/sync-operator.sh

kubectl=./cluster/kubectl.sh

nmstate_cr_manifest=deploy/examples/nmstate.io_v1_nmstate_cr.yaml

function patch_handler_nodeselector() {
    $kubectl patch -f $nmstate_cr_manifest --patch '{"spec": {"nodeSelector": { "node-role.kubernetes.io/worker": "" }}}' --type=merge
}

function wait_ready_nmstate() {
    $kubectl wait --for=condition=Available nmstate/nmstate --timeout=300s
}

sync_operator true
wait_ready_operator true
patch_handler_nodeselector
wait_ready_nmstate
