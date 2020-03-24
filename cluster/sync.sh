#!/bin/bash

set -ex

kubectl=./cluster/kubectl.sh

function getDesiredNumberScheduled {
    $kubectl get daemonset -n nmstate $1 -o=jsonpath='{.status.desiredNumberScheduled}'
}

function getNumberAvailable {
    numberAvailable=$($kubectl get daemonset -n nmstate $1 -o=jsonpath='{.status.numberAvailable}')
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

function isOk {
    desiredNumberScheduled=$(getDesiredNumberScheduled $1)
    numberAvailable=$(getNumberAvailable $1)
    [ "$desiredNumberScheduled" == "$numberAvailable" ]
}

function deploy() {
    # Cleanup previous deployment, if there is any
    make cluster-clean

    # Fetch registry port that can be used to upload images to the local kubevirtci cluster
    registry_port=$(./cluster/cli.sh ports registry | tr -d '\r')
    if [[ "${KUBEVIRT_PROVIDER}" =~ ^(okd|ocp)-.*$ ]]; then \
            registry=localhost:$(./cluster/cli.sh ports --container-name=cluster registry | tr -d '\r')
    else
        registry=localhost:$(./cluster/cli.sh ports registry | tr -d '\r')
    fi

    # Build new handler image from local sources and push it to the kubevirtci cluster
    IMAGE_REGISTRY=${registry} make push-handler

    # Also generate the manifests pointing to the local registry
    IMAGE_REGISTRY=registry:5000 make manifests

    if [[ "$KUBEVIRT_PROVIDER" =~ ^(okd|ocp)-.*$ ]]; then
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
    $kubectl apply -f $MANIFESTS_DIR/nmstate.io_nodenetworkstates_crd.yaml
    $kubectl apply -f $MANIFESTS_DIR/nmstate.io_nodenetworkconfigurationpolicies_crd.yaml
    $kubectl apply -f $MANIFESTS_DIR/nmstate.io_nodenetworkconfigurationenactments_crd.yaml
    $kubectl apply -f $MANIFESTS_DIR/operator.yaml
}

function wait_ready() {
    # Wait until the handler becomes consistently ready on all nodes
    for ds in nmstate-handler nmstate-handler-worker; do
        # We have to re-check desired number, sometimes takes some time to be filled in
        if ! eventually isOk $ds; then
            echo "Daemon set $ds haven't turned ready within the given timeout"
            exit 1
        fi

        # We have to re-check desired number, sometimes takes some time to be filled in
        if ! consistently isOk $ds; then
            echo "Daemon set $ds is not consistently ready within the given timeout"
            exit 1
        fi
    done
}

deploy
wait_ready
