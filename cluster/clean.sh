#!/bin/bash

set -ex

echo 'Cleaning up ...'

MANIFESTS_DIR=build/_output/manifests
kubectl=./cluster/kubectl.sh

if [ ! -d $MANIFESTS_DIR ]; then
    exit 0
fi

$kubectl delete --ignore-not-found -f $MANIFESTS_DIR/operator.yaml
$kubectl delete --ignore-not-found -f $MANIFESTS_DIR/nmstate.io_nodenetworkconfigurationenactments_crd.yaml
$kubectl delete --ignore-not-found -f $MANIFESTS_DIR/nmstate.io_nodenetworkconfigurationpolicies_crd.yaml
$kubectl delete --ignore-not-found -f $MANIFESTS_DIR/nmstate.io_nodenetworkstates_crd.yaml
$kubectl delete --ignore-not-found -f $MANIFESTS_DIR/namespace.yaml
$kubectl delete --ignore-not-found -f $MANIFESTS_DIR/service_account.yaml
$kubectl delete --ignore-not-found -f $MANIFESTS_DIR/role.yaml
$kubectl delete --ignore-not-found -f $MANIFESTS_DIR/role_binding.yaml

if [[ "$KUBEVIRT_PROVIDER" =~ ^(okd|ocp)-.*$ ]]; then
    $kubectl delete --ignore-not-found -f $MANIFESTS_DIR/scc.yaml
fi
