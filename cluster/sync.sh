#!/bin/bash

set -ex

source ./cluster/sync-common.sh
source ./cluster/sync-operator.sh

kubectl=./cluster/kubectl.sh

nmstate_cr_manifest=deploy/examples/nmstate.io_v1_nmstate_cr.yaml

function deploy_handler() {
    $kubectl apply -f $nmstate_cr_manifest
}

function patch_handler_nodeselector() {
    $kubectl patch -f $nmstate_cr_manifest --patch '{"spec": {"nodeSelector": { "node-role.kubernetes.io/worker": "" }}}' --type=merge
}

function patch_handler_backend() {
    local backend=${BACKEND:-nmstate}
    echo "Patching NMState CR with backend: $backend"
    $kubectl patch -f $nmstate_cr_manifest --patch "{\"spec\": {\"backend\": \"$backend\"}}" --type=merge
}

function wait_ready_nmstate() {
    $kubectl wait --for=condition=Available nmstate/nmstate --timeout=300s
}

deploy_operator
wait_ready_operator
deploy_handler
patch_handler_nodeselector
patch_handler_backend
wait_ready_nmstate
