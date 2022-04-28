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

function wait_ready_handler() {
    if ! $kubectl rollout status -w -n ${HANDLER_NAMESPACE} ds nmstate-handler --timeout=5m; then
        echo "Handler haven't turned ready within the given timeout"
        return 1
    fi

    # We have to re-check desired number, sometimes takes some time to be filled in
    if ! $kubectl rollout status -w -n ${HANDLER_NAMESPACE} deployment nmstate-webhook --timeout=5m; then
        echo "Webhook haven't turned ready within the given timeout"
        return 1
    fi
}

deploy_operator
wait_ready_operator
deploy_handler
patch_handler_nodeselector
wait_ready_handler
