#!/bin/bash

set -ex

source ./cluster/lima.sh
lima::ensure_linux

kubectl=./cluster/kubectl.sh
MANIFESTS_DIR=${MANIFESTS_DIR:-build/_output/manifests}
OPERATOR_NAMESPACE=${OPERATOR_NAMESPACE:-nmstate}
HANDLER_NAMESPACE=${HANDLER_NAMESPACE:-nmstate}

source ./cluster/sync-common.sh

function deploy_operator() {
    # Cleanup previous deployment, if there is any
    make cluster-clean

    push

    # Deploy all needed manifests
    $kubectl apply -f $MANIFESTS_DIR/namespace.yaml
    $kubectl apply -f $MANIFESTS_DIR/service_account.yaml
    $kubectl apply -f $MANIFESTS_DIR/role.yaml
    $kubectl apply -f $MANIFESTS_DIR/role_binding.yaml
    $kubectl apply -f deploy/crds/nmstate.io_nmstates.yaml
    $kubectl apply -f $MANIFESTS_DIR/operator.yaml
}

function wait_ready_operator() {
    # Wait a little for resources to be created
    sleep 5

    # Wait for deployment rollout
    if ! $kubectl rollout status -w -n ${OPERATOR_NAMESPACE} deployment nmstate-operator --timeout=2m; then
        echo "Operator haven't turned ready within the given timeout"
        return 1
    fi
}

if [ "$(basename -- $0)" == "sync-operator.sh" ]; then
    deploy_operator
    wait_ready_operator
fi
