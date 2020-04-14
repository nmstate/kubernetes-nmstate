#!/bin/bash

set -ex

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

function clean() {
    echo 'Cleaning up ...'

    MANIFESTS_DIR=build/_output/manifests
    kubectl=./cluster/kubectl.sh

    if [ ! -d $MANIFESTS_DIR ]; then
        exit 0
    fi

    $kubectl delete --ignore-not-found -f deploy/crds/nmstate.io_v1alpha1_nmstate_cr.yaml
    $kubectl delete --ignore-not-found -f $MANIFESTS_DIR/operator.yaml
    $kubectl delete --ignore-not-found -f deploy/crds/nmstate.io_nodenetworkconfigurationenactments_crd.yaml
    $kubectl delete --ignore-not-found -f deploy/crds/nmstate.io_nodenetworkconfigurationpolicies_crd.yaml
    $kubectl delete --ignore-not-found -f deploy/crds/nmstate.io_nodenetworkstates_crd.yaml
    $kubectl delete --ignore-not-found -f deploy/crds/nmstate.io_nmstates_crd.yaml
    $kubectl delete --ignore-not-found -f $MANIFESTS_DIR/namespace.yaml
    $kubectl delete --ignore-not-found -f $MANIFESTS_DIR/service_account.yaml
    $kubectl delete --ignore-not-found -f $MANIFESTS_DIR/role.yaml
    $kubectl delete --ignore-not-found -f $MANIFESTS_DIR/role_binding.yaml

    if [[ "$KUBEVIRT_PROVIDER" =~ ^(okd|ocp)-.*$ ]]; then
        $kubectl delete --ignore-not-found -f $MANIFESTS_DIR/scc.yaml
    fi
}

function isHandlerRemoved {
    $kubectl get daemonset -n ${HANDLER_NAMESPACE} nmstate-handler | grep "NotFound"
}

function wait_removed() {
    if ! eventually isHandlerRemoved; then
        echo "Daemon set nmstate-handler hasn't been removed within the given timeout"
        exit 1
    fi
}

clean
wait_removed
