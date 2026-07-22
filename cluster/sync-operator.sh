#!/bin/bash

set -ex

source ./cluster/lima.sh
lima::ensure_linux

kubectl=./cluster/kubectl.sh
MANIFESTS_DIR=${MANIFESTS_DIR:-build/_output/manifests}
RENDERED_MANIFESTS_DIR=${MANIFESTS_DIR}/kubernetes-nmstate/templates
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
        --set nmstate.enabled=true \
        --set operator.image="${operator_image}" \
        --set operator.pullPolicy="${OPERATOR_PULL_POLICY}" \
        --set handler.image="${handler_image}" \
        --set handler.pullPolicy="${HANDLER_PULL_POLICY}" \
        --set handler.namespace="${HANDLER_NAMESPACE}" \
        --set handler.prefix="${HANDLER_PREFIX:-}" \
        --set monitoring.namespace="${MONITORING_NAMESPACE}" \
        --wait --timeout 5m
}

function clean_operator_manifests() {
    ./cluster/clean.sh
}

function clean_operator_helm() {
    ${HELM} uninstall "${HELM_RELEASE_NAME}" \
        --kubeconfig "${KUBECONFIG}" \
        --namespace "${OPERATOR_NAMESPACE}" \
        --ignore-not-found \
        --wait --timeout 5m
}

function clean_operator() {
    local mode=${1:?operator deployment mode is required}

    case "${mode}" in
    manifests)
        clean_operator_manifests
        ;;
    helm)
        clean_operator_helm
        ;;
    *)
        echo "Unsupported operator deployment mode: ${mode}" >&2
        return 1
        ;;
    esac
}

function deploy_operator() {
    local mode=${1:?operator deployment mode is required}

    # Cleanup previous deployment, if there is any
    if [[ -x "${HELM}" ]]; then
        clean_operator helm
    fi
    clean_operator manifests

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
    case "${2:-deploy}" in
    deploy)
        deploy_operator "${1:?operator deployment mode is required}"
        wait_ready_operator "$1"
        ;;
    clean)
        clean_operator "${1:?operator deployment mode is required}"
        ;;
    *)
        echo "Unsupported sync-operator action: ${2}" >&2
        exit 1
        ;;
    esac
fi
