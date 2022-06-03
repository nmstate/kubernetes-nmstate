
#!/bin/bash

# This is a helper to deploy a bundle to a running cluster (e.g. to validate the
# bundle manifests / csv). This should be called via its make target (`make ocp-build-and-deploy-bundle`)

# Available "parameters": 
#   - IMAGE_REGISTRY (defaults to quay.io)
#   - IMAGE_REPO (defaults to openshift)
#   - HANDLER_IMAGE_NAME (defaults to origin-kubernetes-nmstate-handler)
#   - HANDLER_IMAGE_TAG (defaults to ${CHANNEL})
#   - OPERATOR_IMAGE_NAME (defaults to origin-kubernetes-nmstate-operator)
#   - OPERATOR_IMAGE_TAG (defaults to ${CHANNEL})
#   - CHANNEL (defaults to the latest 4.x version in manifests/)
#   - VERSION (defaults to ${CHANNEL}.0)
#   - BUNDLE_VERSION (defaults to ${VERSION})
#   - INDEX_VERSION (defaults to ${VERSION})

set -ex

if [ -z "${CHANNEL}" ]; then
    # get latest 4.* version from manifests folder
    export CHANNEL=$(find manifests/ -name "4.*" -printf "%f\n" | sort -Vr | head -n 1)
fi

export IMAGE_REGISTRY="${IMAGE_REGISTRY:-quay.io}"
export IMAGE_REPO="${IMAGE_REPO:-openshift}"
export VERSION="${VERSION:-${CHANNEL}.0}"
export NAMESPACE="openshift-nmstate"

export HANDLER_IMAGE_NAME="${HANDLER_IMAGE_NAME:-origin-kubernetes-nmstate-handler}"
export HANDLER_IMAGE_TAG="${HANDLER_IMAGE_TAG:-${VERSION}}"
export HANDLER_NAMESPACE="${NAMESPACE}"

export OPERATOR_IMAGE_NAME="${OPERATOR_IMAGE_NAME:-origin-kubernetes-nmstate-operator}"
export OPERATOR_IMAGE_TAG="${OPERATOR_IMAGE_TAG:-${VERSION}}"
export OPERATOR_NAMESPACE="${NAMESPACE}"

export BUNDLE_VERSION="${BUNDLE_VERSION:-${VERSION}}"
export BUNDLE_IMG="${BUNDLE_IMG:-${IMAGE_REGISTRY}/${IMAGE_REPO}/kubernetes-nmstate-operator-bundle:${BUNDLE_VERSION}}"

export INDEX_VERSION="${INDEX_VERSION:-${VERSION}}"
export INDEX_IMG="${INDEX_IMG:-${IMAGE_REGISTRY}/${IMAGE_REPO}/kubernetes-nmstate-operator-index:${INDEX_VERSION}}"


if [ ! "$SKIP_IMAGE_BUILD" == "true" ]; then
  # create or cleanup tmp dir for bundle manifests to not override manifests in manifests/4.x
  TMP_BUNDLE_DIR=./build/_output/bundle-tmp
  
  if [ -d "${TMP_BUNDLE_DIR}" ]; then
    echo "*** Cleaning up old bundle files from disk... ***"
    rm -rf ${TMP_BUNDLE_DIR}
  fi

  mkdir -p ${TMP_BUNDLE_DIR}

  echo "**** Build and push operator and handler... ****"
  make push-handler push-operator

  echo "**** Create bundle files... ****"
  BUNDLE_DIR=${TMP_BUNDLE_DIR} make ocp-update-bundle-manifests
  # remove the image references file. This leads to issues in "local" deployments
  rm -f ${TMP_BUNDLE_DIR}/manifests/image-references

  echo "**** Build and push bundle... ****"
  BUNDLE_DOCKERFILE="${TMP_BUNDLE_DIR}/bundle.Dockerfile" make bundle-build bundle-push

  echo "**** Build and push index... ****"
  BUNDLE_DOCKERFILE="${TMP_BUNDLE_DIR}/bundle.Dockerfile" make index-build index-push
fi

echo "**** Create catalog source ****"
cat <<EOF | oc create -f - 
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: kubernetes-nmstate-catalog
  namespace: openshift-marketplace
spec:
  sourceType: grpc
  image: ${INDEX_IMG}
  displayName: Catalog for kubernetes-nmstate
  publisher: knmstate-catalog
EOF

if [ "$INSTALL_OPERATOR_VIA_UI" == "true" ]; then
  echo "**** Skipping installing operator. Has to be installed via console UI ****"
  exit
fi

echo "**** Create namespace if it does not exist ****"
oc create namespace "${OPERATOR_NAMESPACE}" --dry-run=client -o yaml | oc apply -f -

echo "**** Create operator group ****"
cat <<EOF | oc create -f - 
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: openshift-kubernetes-nmstate-operator
  namespace: ${OPERATOR_NAMESPACE}
spec:
  targetNamespaces:
  - ${OPERATOR_NAMESPACE}
EOF

echo "**** Create subscription ****"
cat <<EOF | oc create -f - 
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: kubernetes-nmstate-operator
  namespace: ${OPERATOR_NAMESPACE}
spec:
  channel: "${CHANNEL}"
  installPlanApproval: Automatic
  name: kubernetes-nmstate-operator
  source: kubernetes-nmstate-catalog
  sourceNamespace: openshift-marketplace
EOF

echo "**** Waiting for install plan to finish ****"
oc -n ${OPERATOR_NAMESPACE} wait --for=condition=installplanpending subscription kubernetes-nmstate-operator
install_plan=$(oc -n ${OPERATOR_NAMESPACE} get subscription kubernetes-nmstate-operator -ojsonpath='{..status.installPlanRef.name}')
oc -n ${OPERATOR_NAMESPACE} wait --for=condition=installed --timeout 120s installplan ${install_plan}

echo "**** Waiting for operator deployment being available ****"
oc -n ${OPERATOR_NAMESPACE} wait --for=condition=available deploy nmstate-operator
