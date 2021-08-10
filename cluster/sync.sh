#!/bin/bash

set -ex

source ./cluster/sync-common.sh
source ./cluster/sync-operator.sh

kubectl=./cluster/kubectl.sh

nmstate_cr_manifest=deploy/crds/nmstate.io_v1beta1_nmstate_cr.yaml

function deploy_handler() {
    $kubectl apply -f $nmstate_cr_manifest
}

function patch_handler_nodeselector() {
    $kubectl patch -f $nmstate_cr_manifest --patch '{"spec": {"nodeSelector": { "node-role.kubernetes.io/worker": "" }}}' --type=merge
}

function wait_ready_handler() {
    # We have to re-check desired number, sometimes takes some time to be filled in
    if ! eventually isDaemonSetOk ${HANDLER_NAMESPACE} app=kubernetes-nmstate ; then
        echo "Handler haven't turned ready within the given timeout"
        return 1
    fi

    # Make sure good state is keep for some time
    if ! consistently isDaemonSetOk ${HANDLER_NAMESPACE} app=kubernetes-nmstate ; then
        echo "Handler is not consistently ready within the given timeout"
        return 1
    fi

    # We have to re-check desired number, sometimes takes some time to be filled in
    if ! eventually isDeploymentOk ${HANDLER_NAMESPACE} app=kubernetes-nmstate; then
        echo "Webhook haven't turned ready within the given timeout"
        return 1
    fi

    # Make sure good state is keep for some time
    if ! consistently isDeploymentOk ${HANDLER_NAMESPACE} app=kubernetes-nmstate; then
        echo "Webhook is not consistently ready within the given timeout"
        return 1
    fi
}

deploy_operator
wait_ready_operator
deploy_handler
patch_handler_nodeselector
wait_ready_handler
