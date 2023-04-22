#!/bin/bash

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
    export CHANNEL="stable"
fi

export BUNDLE_DIR="${BUNDLE_DIR:-manifests/${CHANNEL}}"
MANIFEST_BASES_DIR=manifests/bases

if [ -z "${VERSION}" ]; then
    export VERSION="$($(yq4) '.spec.version' ${BUNDLE_DIR}/manifests/kubernetes-nmstate-operator.clusterserviceversion.yaml)"
fi
export VERSION_MAJOR_MINOR="${VERSION%.*}"

export IMAGE_REGISTRY="${IMAGE_REGISTRY:-quay.io}"
export IMAGE_REPO="${IMAGE_REPO:-openshift}"
export NAMESPACE="openshift-nmstate"

export HANDLER_IMAGE_NAME="${HANDLER_IMAGE_NAME:-origin-kubernetes-nmstate-handler}"
export HANDLER_IMAGE_TAG="${HANDLER_IMAGE_TAG:-${VERSION_MAJOR_MINOR}}" # TODO(chocobomb) Do we tag as "stable" or still "4.x" ?
export HANDLER_NAMESPACE="${NAMESPACE}"

export OPERATOR_IMAGE_NAME="${OPERATOR_IMAGE_NAME:-origin-kubernetes-nmstate-operator}"
export OPERATOR_IMAGE_TAG="${OPERATOR_IMAGE_TAG:-${VERSION_MAJOR_MINOR}}" # TODO(chocobomb) Do we tag as "stable" or still "4.x" ?
export OPERATOR_NAMESPACE="${NAMESPACE}"

export PLUGIN_IMAGE_NAME="${PLUGIN_IMAGE_NAME:-origin-nmstate-console-plugin}"
export PLUGIN_IMAGE_TAG="${PLUGIN_IMAGE_TAG:-${VERSION_MAJOR_MINOR}}" # TODO(chocobomb) Do we tag as "stable" or still "4.x" ?
export PLUGIN_NAMESPACE="${NAMESPACE}"

export BUNDLE_VERSION="${BUNDLE_VERSION:-${VERSION_MAJOR_MINOR}}" # TODO(chocobomb) X.Y or X.Y.Z here? Is this variable even used?
export BUNDLE_IMG="${BUNDLE_IMG:-${IMAGE_REGISTRY}/${IMAGE_REPO}/kubernetes-nmstate-operator-bundle:${BUNDLE_VERSION}}" # TODO(chocobomb) Is this variable even used?

export INDEX_VERSION="${INDEX_VERSION:-${VERSION_MAJOR_MINOR}}" # TODO(chocobomb) X.Y or X.Y.Z here?
export INDEX_IMG="${INDEX_IMG:-${IMAGE_REGISTRY}/${IMAGE_REPO}/kubernetes-nmstate-operator-index:${INDEX_VERSION}}"
