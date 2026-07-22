#!/bin/bash

set -ex

source ./cluster/lima.sh
lima::ensure_linux

kubectl=./cluster/kubectl.sh
OPERATOR_NAMESPACE=${OPERATOR_NAMESPACE:-nmstate}
HANDLER_NAMESPACE=${HANDLER_NAMESPACE:-nmstate}

source ./cluster/sync-common.sh

function deploy_operator_helm() {
    # Cleanup previous deployment, if there is any
    make cluster-clean

    # push() builds and pushes images. On kubevirtci providers it also exports
    # IMAGE_REGISTRY / OPERATOR_IMAGE_FULL_NAME / HANDLER_IMAGE_FULL_NAME.
    push

    if isExternal; then
        make IMAGE_REGISTRY=${DEV_IMAGE_REGISTRY} OPERATOR_NAMESPACE=${OPERATOR_NAMESPACE} HANDLER_NAMESPACE=${HANDLER_NAMESPACE} helm-install
    else
        make OPERATOR_NAMESPACE=${OPERATOR_NAMESPACE} HANDLER_NAMESPACE=${HANDLER_NAMESPACE} helm-install
    fi
}

function nns_exist() {
    [[ -n "$($kubectl get nns -o name 2>/dev/null)" ]]
}

function wait_ready_handler() {
    # The operator deploys the handler in reaction to the chart-created
    # NMState CR (nmstate.enabled=true)
    if ! eventually $kubectl rollout status -w -n ${HANDLER_NAMESPACE} ds "${HANDLER_PREFIX:-}nmstate-handler" --timeout=2m; then
        echo "Handler hasn't turned ready within the given timeout"
        return 1
    fi
    eventually nns_exist
}

if [ "$(basename -- $0)" == "sync-operator-helm.sh" ]; then
    deploy_operator_helm
    wait_ready_handler
fi
