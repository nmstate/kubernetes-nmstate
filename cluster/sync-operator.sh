#!/bin/bash

set -ex

source ./cluster/lima.sh
lima::ensure_linux

kubectl=./cluster/kubectl.sh
OPERATOR_NAMESPACE=${OPERATOR_NAMESPACE:-nmstate}
HANDLER_NAMESPACE=${HANDLER_NAMESPACE:-nmstate}
HELM_VERSION=${HELM_VERSION:-v3.16.2}
HELM=${HELM:-./build/_output/bin/helm-${HELM_VERSION}}
HELM_RELEASE_NAME=${HELM_RELEASE_NAME:-nmstate}
MONITORING_NAMESPACE=${MONITORING_NAMESPACE:-monitoring}
IMAGE_REPO=${IMAGE_REPO:-nmstate}
IMAGE_REGISTRY=${IMAGE_REGISTRY:-quay.io}
OPERATOR_IMAGE_NAME=${OPERATOR_IMAGE_NAME:-kubernetes-nmstate-operator}
OPERATOR_IMAGE_TAG=${OPERATOR_IMAGE_TAG:-latest}
OPERATOR_IMAGE_FULL_NAME=${OPERATOR_IMAGE_FULL_NAME:-${IMAGE_REPO}/${OPERATOR_IMAGE_NAME}:${OPERATOR_IMAGE_TAG}}
HANDLER_IMAGE_NAME=${HANDLER_IMAGE_NAME:-kubernetes-nmstate-handler}
HANDLER_IMAGE_TAG=${HANDLER_IMAGE_TAG:-latest}
HANDLER_IMAGE_FULL_NAME=${HANDLER_IMAGE_FULL_NAME:-${IMAGE_REPO}/${HANDLER_IMAGE_NAME}:${HANDLER_IMAGE_TAG}}

if [[ -t 0 ]]; then
    OPERATOR_PULL_POLICY=${OPERATOR_PULL_POLICY:-Always}
    HANDLER_PULL_POLICY=${HANDLER_PULL_POLICY:-Always}
else
    OPERATOR_PULL_POLICY=${OPERATOR_PULL_POLICY:-IfNotPresent}
    HANDLER_PULL_POLICY=${HANDLER_PULL_POLICY:-IfNotPresent}
fi

source ./cluster/sync-common.sh

function deploy_operator() {
    local nmstate_enabled=${1:-false}
    local operator_image=${IMAGE_REGISTRY}/${OPERATOR_IMAGE_FULL_NAME}
    local handler_image=${IMAGE_REGISTRY}/${HANDLER_IMAGE_FULL_NAME}

    if isExternal; then
        operator_image=${DEV_IMAGE_REGISTRY}/${OPERATOR_IMAGE_FULL_NAME}
        handler_image=${DEV_IMAGE_REGISTRY}/${HANDLER_IMAGE_FULL_NAME}
    fi

    ${HELM} upgrade --install "${HELM_RELEASE_NAME}" charts/kubernetes-nmstate \
        --kubeconfig "${KUBECONFIG}" \
        --namespace "${OPERATOR_NAMESPACE}" \
        --create-namespace \
        --set nmstate.enabled="${nmstate_enabled}" \
        --set operator.image="${operator_image}" \
        --set operator.pullPolicy="${OPERATOR_PULL_POLICY}" \
        --set handler.image="${handler_image}" \
        --set handler.pullPolicy="${HANDLER_PULL_POLICY}" \
        --set handler.namespace="${HANDLER_NAMESPACE}" \
        --set handler.prefix="${HANDLER_PREFIX:-}" \
        --set monitoring.namespace="${MONITORING_NAMESPACE}" \
        --wait --timeout 5m
}

function clean_operator() {
    ${HELM} uninstall "${HELM_RELEASE_NAME}" \
        --kubeconfig "${KUBECONFIG}" \
        --namespace "${OPERATOR_NAMESPACE}" \
        --ignore-not-found \
        --wait --timeout 5m
}

function sync_operator() {
    local nmstate_enabled=${1:-false}
    # Cleanup previous deployment, if there is any
    if [[ -x "${HELM}" ]]; then
        clean_operator
    fi

    # push() builds and pushes images. On kubevirtci providers it also exports
    # IMAGE_REGISTRY / OPERATOR_IMAGE_FULL_NAME / HANDLER_IMAGE_FULL_NAME.
    push

    deploy_operator "${nmstate_enabled}"
}

function nns_exist() {
    [[ -n "$($kubectl get nns -o name 2>/dev/null)" ]]
}

function wait_ready_operator() {
    local nmstate_enabled=${1:-false}

    # Wait a little for resources to be created
    sleep 5

    # Wait for deployment rollout
    if ! $kubectl rollout status -w -n ${OPERATOR_NAMESPACE} deployment nmstate-operator --timeout=2m; then
        echo "Operator haven't turned ready within the given timeout"
        return 1
    fi

    if [[ "${nmstate_enabled}" != "true" ]]; then
        return 0
    fi

    # The operator deploys the handler in reaction to the chart-created
    # NMState CR (nmstate.enabled=true)
    if ! eventually $kubectl rollout status -w -n ${HANDLER_NAMESPACE} ds "${HANDLER_PREFIX:-}nmstate-handler" --timeout=2m; then
        echo "Handler hasn't turned ready within the given timeout"
        return 1
    fi
    eventually nns_exist
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    case "${1:-deploy}" in
    deploy)
        sync_operator
        wait_ready_operator
        ;;
    deploy-with-nmstate)
        sync_operator true
        wait_ready_operator true
        ;;
    clean)
        clean_operator
        ;;
    *)
        echo "Unsupported sync-operator action: ${1}" >&2
        exit 1
        ;;
    esac
fi
