#!/bin/bash

# This script is an automation of steps 1-7 from
# https://hackmd.io/0vbbJdZ7RXem69vCWI61ug to install the nightly ART bundle
# build of the OpenShift operators.  The script requires the user to have a
# employee-token created at least once before
# (https://source.redhat.com/groups/public/teamnado/wiki/brew_registry).

set -eo pipefail

function wait_for_all_nodes_ready() {
  i=20
  while [ $i -gt 0 ]; do
    if [[ $(oc get no | grep SchedulingDisabled) ]]; then
      # reset timer
      i=20
      echo "found node with SchedulingDisabled. Resetting timer..."
    else
      echo "Waiting ${i} times to ensure all nodes are ready..."
      i=$((i - 1))
    fi

    sleep 1
  done
}	

brew_credentials=$(curl --negotiate -u : https://employee-token-manager.registry.redhat.com/v1/tokens -s) #make sure you created a token before (https://source.redhat.com/groups/public/teamnado/wiki/brew_registry)
if [[ "${brew_credentials}" == "null" ]]; then
  echo "No Brew token found. Trying to create a token..."
  # Brew token does not exist. Create one
  brew_credentials=$(curl --negotiate -u : -X POST -H 'Content-Type: application/json' --data '{"description":"openshift 4 testing"}' https://employee-token-manager.registry.redhat.com/v1/tokens -s)
  if [[ -z "$brew_credentials" ]] || [[ "$brew_credentials" == "null" ]]; then
    echo "Could not create a Brew token for you. Please visit https://source.redhat.com/groups/public/teamnado/wiki/brew_registry for more information about creating the Brew token manually." 
    exit 1
  fi
  echo "Brew token created"
  brew_credentials="["${brew_credentials}"]" # convert to json array
fi

brew_username=$(echo $brew_credentials | jq -r ".[0].credentials.username")
brew_password=$(echo $brew_credentials | jq -r ".[0].credentials.password")

cluster_version=$(oc get clusterversion version -o jsonpath='{.status.desired.version}' | cut -d '.' -f 1,2)

oc patch OperatorHub cluster --type json -p '[{"op": "add", "path": "/spec/disableAllDefaultSources","value": true}]'
oc get secret/pull-secret -n openshift-config -o json | jq -r '.data.".dockerconfigjson"' | base64 -d > authfile
if ! podman login --authfile authfile --username "${brew_username}" --password "${brew_password}" brew.registry.redhat.io; then
  rm authfile # make sure the authfile is deleted in case of an error
  exit 1
fi

if ! oc set data secret/pull-secret -n openshift-config --from-file=.dockerconfigjson=authfile; then
  rm authfile # make sure the authfile is deleted in case of an error
  exit 1
fi

rm authfile
oc patch image.config.openshift.io/cluster --type merge -p '{"spec":{"registrySources":{"insecureRegistries":["registry-proxy.engineering.redhat.com"]}}}'

wait_for_all_nodes_ready

cat <<EOF | oc apply -f -
apiVersion: operator.openshift.io/v1alpha1
kind: ImageContentSourcePolicy
metadata:
  name: brew-registry
spec:
  repositoryDigestMirrors:
  - mirrors:
    - brew.registry.redhat.io
    source: registry.redhat.io
  - mirrors:
    - brew.registry.redhat.io
    source: registry.stage.redhat.io
  - mirrors:
    - brew.registry.redhat.io
    source: registry-proxy.engineering.redhat.com
EOF

wait_for_all_nodes_ready

cat <<EOF | oc apply -f -
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: my-operator-catalog
  namespace: openshift-marketplace
spec:
  sourceType: grpc
  image: quay.io/openshift-release-dev/ocp-release-nightly:iib-int-index-art-operators-${cluster_version}
  displayName: My Operator Catalog
  publisher: grpc
EOF
