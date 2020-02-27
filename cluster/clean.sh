#!/bin/bash

set -ex

echo 'Cleaning up ...'

manifests_dir=build/_output/manifests
kubectl=./cluster/kubectl.sh

if [ ! -d $manifests_dir ]; then
    exit 0
fi

$kubectl delete --ignore-not-found -f $manifests_dir/operator.yaml
$kubectl delete --ignore-not-found -f $manifests_dir/nmstate.io_nodenetworkconfigurationenactments_crd.yaml
$kubectl delete --ignore-not-found -f $manifests_dir/nmstate.io_nodenetworkconfigurationpolicies_crd.yaml
$kubectl delete --ignore-not-found -f $manifests_dir/nmstate.io_nodenetworkstates_crd.yaml
$kubectl delete --ignore-not-found -f $manifests_dir/namespace.yaml
$kubectl delete --ignore-not-found -f $manifests_dir/service_account.yaml
$kubectl delete --ignore-not-found -f $manifests_dir/role.yaml
$kubectl delete --ignore-not-found -f $manifests_dir/role_binding.yaml

if [[ "$KUBEVIRT_PROVIDER" =~ ^(okd|ocp)-.*$ ]]; then
    $kubectl delete --ignore-not-found -f $manifests_dir/scc.yaml
fi
