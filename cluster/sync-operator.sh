#!/bin/bash

set -ex

kubectl=./cluster/kubectl.sh

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
    # We have to re-check desired number, sometimes takes some time to be filled in
    if ! eventually isDeploymentOk ${OPERATOR_NAMESPACE} app=kubernetes-nmstate-operator; then
        echo "Operator haven't turned ready within the given timeout"
        return 1
    fi

    # Make sure good state is keep for some time
    if ! consistently isDeploymentOk ${OPERATOR_NAMESPACE} app=kubernetes-nmstate-operator; then
        echo "Operator is not consistently ready within the given timeout"
        return 1
    fi
}

if [ "$(basename -- $0)" == "sync-operator.sh" ]; then
    deploy_operator
    wait_ready_operator
fi
