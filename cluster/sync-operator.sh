#!/bin/bash

set -ex

source ./cluster/lima.sh
lima::ensure_linux

kubectl=./cluster/kubectl.sh
MANIFESTS_DIR=${MANIFESTS_DIR:-build/_output/manifests}
RENDERED_MANIFESTS_DIR=${MANIFESTS_DIR}/kubernetes-nmstate/templates
OPERATOR_NAMESPACE=${OPERATOR_NAMESPACE:-nmstate}
HANDLER_NAMESPACE=${HANDLER_NAMESPACE:-nmstate}

source ./cluster/sync-common.sh

function deploy_operator_manifests() {
    # Deploy all needed manifests
    $kubectl apply -f $RENDERED_MANIFESTS_DIR/namespace.yaml
    $kubectl apply -f $RENDERED_MANIFESTS_DIR/service_account.yaml
    $kubectl apply -f $RENDERED_MANIFESTS_DIR/role.yaml
    $kubectl apply -f $RENDERED_MANIFESTS_DIR/role_binding.yaml
    $kubectl apply -f deploy/crds/nmstate.io_nmstates.yaml
    $kubectl apply -f $RENDERED_MANIFESTS_DIR/operator.yaml
}

function deploy_operator_helm() {
    if isExternal; then
        make IMAGE_REGISTRY=${DEV_IMAGE_REGISTRY} OPERATOR_NAMESPACE=${OPERATOR_NAMESPACE} HANDLER_NAMESPACE=${HANDLER_NAMESPACE} helm-install
    else
        make OPERATOR_NAMESPACE=${OPERATOR_NAMESPACE} HANDLER_NAMESPACE=${HANDLER_NAMESPACE} helm-install
    fi
}

function deploy_operator() {
    local mode=${1:?operator deployment mode is required}

    # Cleanup previous deployment, if there is any
    make cluster-clean

    # push() builds and pushes images. On kubevirtci providers it also exports
    # IMAGE_REGISTRY / OPERATOR_IMAGE_FULL_NAME / HANDLER_IMAGE_FULL_NAME.
    push

    case "${mode}" in
    manifests)
        deploy_operator_manifests
        ;;
    helm)
        deploy_operator_helm
        ;;
    *)
        echo "Unsupported operator deployment mode: ${mode}" >&2
        return 1
        ;;
    esac
}

function nns_exist() {
    [[ -n "$($kubectl get nns -o name 2>/dev/null)" ]]
}

function wait_ready_operator_manifests() {
    # Wait a little for resources to be created
    sleep 5

    # Wait for deployment rollout
    if ! $kubectl rollout status -w -n ${OPERATOR_NAMESPACE} deployment nmstate-operator --timeout=2m; then
        echo "Operator haven't turned ready within the given timeout"
        return 1
    fi
}

function wait_ready_operator_helm() {
    # The operator deploys the handler in reaction to the chart-created
    # NMState CR (nmstate.enabled=true)
    if ! eventually $kubectl rollout status -w -n ${HANDLER_NAMESPACE} ds "${HANDLER_PREFIX:-}nmstate-handler" --timeout=2m; then
        echo "Handler hasn't turned ready within the given timeout"
        return 1
    fi
    eventually nns_exist
}

function wait_ready_operator() {
    local mode=${1:?operator deployment mode is required}

    case "${mode}" in
    manifests)
        wait_ready_operator_manifests
        ;;
    helm)
        wait_ready_operator_helm
        ;;
    *)
        echo "Unsupported operator deployment mode: ${mode}" >&2
        return 1
        ;;
    esac
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    deploy_operator "${1:?operator deployment mode is required}"
    wait_ready_operator "$1"
fi
