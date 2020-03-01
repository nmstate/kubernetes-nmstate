#!/bin/bash

set -ex

echo 'Cleaning up ...'

./cluster/kubectl.sh delete --ignore-not-found -f deploy/
./cluster/kubectl.sh delete --ignore-not-found -f deploy/crds/nmstate.io_nodenetworkconfigurationenactments_crd.yaml
./cluster/kubectl.sh delete --ignore-not-found -f deploy/crds/nmstate.io_nodenetworkconfigurationpolicies_crd.yaml
./cluster/kubectl.sh delete --ignore-not-found -f deploy/crds/nmstate.io_nodenetworkstates_crd.yaml

if [[ "$KUBEVIRT_PROVIDER" =~ ^(okd|ocp)-.*$ ]]; then
    ./cluster/kubectl.sh delete --ignore-not-found -f deploy/openshift/
fi
