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

    # Delete the CR only if the CRD is installed otherwise it will fail
    if $kubectl get crds nmstates.nmstate.io; then
        $kubectl delete --ignore-not-found -f deploy/crds/nmstate.io_v1beta1_nmstate_cr.yaml
    fi
    $kubectl delete --ignore-not-found -f $MANIFESTS_DIR/operator.yaml
    $kubectl delete --ignore-not-found -f deploy/crds/nmstate.io_nodenetworkconfigurationenactments.yaml
    $kubectl delete --ignore-not-found -f deploy/crds/nmstate.io_nodenetworkconfigurationpolicies.yaml
    $kubectl delete --ignore-not-found -f deploy/crds/nmstate.io_nodenetworkstates.yaml
    $kubectl delete --ignore-not-found -f deploy/crds/nmstate.io_nmstates.yaml
    $kubectl delete --ignore-not-found -f $MANIFESTS_DIR/namespace.yaml
    $kubectl delete --ignore-not-found -f $MANIFESTS_DIR/service_account.yaml
    $kubectl delete --ignore-not-found -f $MANIFESTS_DIR/role.yaml
    $kubectl delete --ignore-not-found -f $MANIFESTS_DIR/role_binding.yaml

    if [[ "$KUBEVIRT_PROVIDER" =~ ^(okd|ocp)-.*$ ]]; then
        $kubectl delete --ignore-not-found -f $MANIFESTS_DIR/scc.yaml
    fi
}

# Use labels so we don't care about prefixes
function isRemoved {
    output=$($kubectl get $1 -n $2 -l $3 2>&1)
    [[ ! $output =~ ".*No resources found.*" ]]
}

function isHandlerRemoved {
    isRemoved daemonset ${HANDLER_NAMESPACE} app=kubernetes-nmstate
}

function isWebhookRemoved {
    isRemoved deployment ${HANDLER_NAMESPACE} app=kubernetes-nmstate
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
