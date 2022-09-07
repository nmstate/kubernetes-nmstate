#!/bin/bash

# This is a helper to update the bundles manifests file. This should be invoked
# via its make target (`make ocp-update-bundle-manifests`)

set -ex

function yq4 {
  VERSION_REGEX=" version 4\.[0-9]+\.[0-9]$"
  if [[ "`yq --version`" =~ $VERSION_REGEX ]]; then
    # installed yq version is v4 -> we are OK
    echo yq
  else
    # version from yq in path is != 4 --> check for alternatives
    INSTALL_DIR=$(pwd)/build/_output/bin
    if [[ -f ${INSTALL_DIR}/yq ]] && [[ "`${INSTALL_DIR}/yq --version`" =~ $VERSION_REGEX ]]; then
      # yq is installed at a 2nd location already and in the correct version --> nothing to do
      echo ${INSTALL_DIR}/yq
    else
      # yq is not installed/in wrong version --> install v4
      GOBIN=$INSTALL_DIR GOFLAGS= go install github.com/mikefarah/yq/v4@latest
      echo ${INSTALL_DIR}/yq
    fi
  fi
}

if [ -z "${CHANNEL}" ]; then
    export CHANNEL=$(find manifests/ -name "4.*" -printf "%f\n" | sort -Vr | head -n 1)
fi

export IMAGE_REPO="${IMAGE_REPO:-openshift}"
export NAMESPACE="openshift-nmstate"

export HANDLER_IMAGE_NAME="${HANDLER_IMAGE_NAME:-origin-kubernetes-nmstate-handler}"
export HANDLER_IMAGE_TAG="${HANDLER_IMAGE_TAG:-${CHANNEL}}"
export HANDLER_NAMESPACE="${NAMESPACE}"

export OPERATOR_IMAGE_NAME="${OPERATOR_IMAGE_NAME:-origin-kubernetes-nmstate-operator}"
export OPERATOR_IMAGE_TAG="${OPERATOR_IMAGE_TAG:-${CHANNEL}}"
export OPERATOR_NAMESPACE="${NAMESPACE}"

export VERSION="${VERSION:-${CHANNEL}.0}"

export BUNDLE_DIR="${BUNDLE_DIR:-manifests/${CHANNEL}}"
MANIFEST_BASES_DIR=manifests/bases

# remove old manifests & bundle metadata files
rm -rf ${BUNDLE_DIR}/manifests ${BUNDLE_DIR}/metadata

# generate bundle files from scratch
IMAGE_REPO=${IMAGE_REPO} \
HANDLER_IMAGE_NAME=${HANDLER_IMAGE_NAME} HANDLER_IMAGE_TAG=${HANDLER_IMAGE_TAG} HANDLER_NAMESPACE=${HANDLER_NAMESPACE} \
OPERATOR_IMAGE_NAME=${OPERATOR_IMAGE_NAME} OPERATOR_IMAGE_TAG=${OPERATOR_IMAGE_TAG} OPERATOR_NAMESPACE=${OPERATOR_NAMESPACE} \
VERSION=${VERSION} CHANNELS=${CHANNEL},alpha DEFAULT_CHANNEL=${CHANNEL} \
BUNDLE_DIR=${BUNDLE_DIR} MANIFEST_BASES_DIR=${MANIFEST_BASES_DIR} make bundle

# add the cluster permissions to use the privileged security context constraint to the nmstate-operator SA in the CSV
$(yq4) --inplace eval '.spec.install.spec.clusterPermissions[] |= select(.rules[]) |= select(.serviceAccountName == "nmstate-operator").rules += {"apiGroups":["security.openshift.io"],"resources":["securitycontextconstraints"],"verbs":["use"],"resourceNames":["privileged"]}' ${BUNDLE_DIR}/manifests/kubernetes-nmstate-operator.clusterserviceversion.yaml

# remove unneeded owned CRDs in CSV / use only NMState v1 CRD
$(yq4) --inplace eval '.spec.customresourcedefinitions.owned |= [{"kind":"NMState","name":"nmstates.nmstate.io","version":"v1","description":"Represents an NMState deployment.","displayName":"NMState"}]' ${BUNDLE_DIR}/manifests/kubernetes-nmstate-operator.clusterserviceversion.yaml

# delete unneeded files
rm -f ${BUNDLE_DIR}/manifests/nmstate.io_nodenetwork*.yaml

# save new bundle.Dockerfile with new paths
sed 's#manifests\/$(CHANNEL)/##g' bundle.Dockerfile > ${BUNDLE_DIR}/bundle.Dockerfile

# save image-refences file
cat > ${BUNDLE_DIR}/manifests/image-references <<EOF
kind: ImageStream
apiVersion: image.openshift.io/v1
spec:
  tags:
  - name: kubernetes-nmstate-operator
    from:
      kind: DockerImage
      name: quay.io/openshift/origin-kubernetes-nmstate-operator:${CHANNEL}
  - name: kubernetes-nmstate-handler
    from:
      kind: DockerImage
      name: quay.io/openshift/origin-kubernetes-nmstate-handler:${CHANNEL}
EOF

# undo changes on "root" bundle.Dockerfile (gets updated by `make bundle`)
git checkout bundle.Dockerfile
