#!/bin/bash

set -ex

kubectl=./cluster/kubectl.sh

function getDesiredScheduledHandlers {
    $kubectl get daemonset -n ${HANDLER_NAMESPACE} $1 -o=jsonpath='{.status.desiredNumberScheduled}'
}

function getAvailableHandlers {
    numberAvailable=$($kubectl get daemonset -n ${HANDLER_NAMESPACE} $1 -o=jsonpath='{.status.numberAvailable}')
    echo ${numberAvailable:-0}
}

function eventually {
    timeout=15
    interval=5
    cmd=$@
    echo "Checking eventually $cmd"
    while ! $cmd; do
        sleep $interval
        timeout=$(( $timeout - $interval ))
        if [ $timeout -le 0 ]; then
            return 1
        fi
    done
}

function consistently {
    timeout=15
    interval=5
    cmd=$@
    echo "Checking consistently $cmd"
    while $cmd; do
        sleep $interval
        timeout=$(( $timeout - $interval ))
        if [ $timeout -le 0 ]; then
            return 0
        fi
    done
}

function isHandlerOk {
    desiredNumberScheduled=$(getDesiredScheduledHandlers $1)
    numberAvailable=$(getAvailableHandlers $1)
    [ "$desiredNumberScheduled" == "$numberAvailable" ]
}

function isExternal {
    [[ "${KUBEVIRT_PROVIDER}" == external ]]
}

function isOpenshift {
    $kubectl get co openshift-apiserver
}

function push() {
    if isExternal; then
        make manifests push
        return 0
    fi
    # Fetch registry port that can be used to upload images to the local kubevirtci cluster
    registry_port=$(./cluster/cli.sh ports registry | tr -d '\r')
    if [[ "${KUBEVIRT_PROVIDER}" =~ ^(okd|ocp)-.*$ ]]; then \
            registry=localhost:$(./cluster/cli.sh ports --container-name=cluster registry | tr -d '\r')
    else
        registry=localhost:$(./cluster/cli.sh ports registry | tr -d '\r')
    fi

    # Build new handler and operator image from local sources and push it to the kubevirtci cluster
    IMAGE_REGISTRY=${registry} make push

    # Also generate the manifests pointing to the local registry
    IMAGE_REGISTRY=registry:5000 make manifests
}

function deploy_operator() {
    # Cleanup previous deployment, if there is any
    make cluster-clean

    push

    if isOpenshift; then
        while ! $kubectl get securitycontextconstraints; do
            sleep 1
        done
        $kubectl apply -f ${MANIFESTS_DIR}/scc.yaml
    fi


    # Deploy all needed manifests
    $kubectl apply -f $MANIFESTS_DIR/namespace.yaml
    $kubectl apply -f $MANIFESTS_DIR/service_account.yaml
    $kubectl apply -f $MANIFESTS_DIR/role.yaml
    $kubectl apply -f $MANIFESTS_DIR/role_binding.yaml
    $kubectl apply -f deploy/crds/nmstate.io_nmstates_crd.yaml
    $kubectl apply -f $MANIFESTS_DIR/operator.yaml
}

function deploy_handler() {
    $kubectl apply -f deploy/crds/nmstate.io_v1alpha1_nmstate_cr.yaml
}

function wait_ready_handler() {
    # Wait until the operator becomes consistently ready on all nodes
    for ds in nmstate-handler nmstate-handler-worker; do
        # We have to re-check desired number, sometimes takes some time to be filled in
        if ! eventually isHandlerOk $ds; then
            echo "Daemon set $ds haven't turned ready within the given timeout"
            exit 1
        fi

        # We have to re-check desired number, sometimes takes some time to be filled in
        if ! consistently isHandlerOk $ds; then
            echo "Daemon set $ds is not consistently ready within the given timeout"
            exit 1
        fi
    done
}

function wait_ready_operator() {
    $kubectl wait deployment -n ${OPERATOR_NAMESPACE} -l app=kubernetes-nmstate-operator --for condition=Available --timeout=200s
}

deploy_operator
wait_ready_operator
deploy_handler
wait_ready_handler
