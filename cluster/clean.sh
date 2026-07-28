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

function eventually {
    timeout=15
    interval=5
    cmd=$@
    echo "Checking eventually $cmd"
    while ! $cmd; do
        if [ $timeout -le 0 ]; then
            return 1
        fi
        sleep $interval
        timeout=$(( $timeout - $interval ))
    done
}

function clean() {
    echo 'Cleaning up ...'

    clean_operator
}

function clean_operator() {
    ${HELM} uninstall "${HELM_RELEASE_NAME}" \
        --kubeconfig "${KUBECONFIG}" \
        --namespace "${OPERATOR_NAMESPACE}" \
        --ignore-not-found \
        --wait --timeout 5m
}

# Use labels so we don't care about prefixes
function isRemoved {
    output=$($kubectl get $1 -n $2 -l $3 2>&1)
    [[ ! $output =~ ".*No resources found.*" ]]
}

function isHandlerRemoved {
    isRemoved daemonset ${HANDLER_NAMESPACE:-nmstate} app=kubernetes-nmstate
}

function isWebhookRemoved {
    isRemoved deployment ${HANDLER_NAMESPACE:-nmstate} app=kubernetes-nmstate
}

function wait_removed() {
    if ! eventually isHandlerRemoved; then
        echo "Handler hasn't been removed within the given timeout"
        exit 1
    fi

    if ! eventually isWebhookRemoved; then
        echo "Webhook hasn't been removed within the given timeout"
        exit 1
    fi

}

clean
wait_removed
