#!/bin/bash

set -ex

echo 'Cleaning up ...'

kubectl=./cluster/kubectl.sh
manifests_dir=deploy

$kubectl delete --ignore-not-found -f $manifests_dir/
$kubectl delete --ignore-not-found -f $manifests_dir/crds/nmstate.io_nodenetworkconfigurationenactments_crd.yaml
$kubectl delete --ignore-not-found -f $manifests_dir/crds/nmstate.io_nodenetworkconfigurationpolicies_crd.yaml
$kubectl delete --ignore-not-found -f $manifests_dir/crds/nmstate.io_nodenetworkstates_crd.yaml

if [[ "$KUBEVIRT_PROVIDER" =~ ^(okd|ocp)-.*$ ]]; then
    $kubectl delete --ignore-not-found -f $manifests_dir/openshift/
fi
