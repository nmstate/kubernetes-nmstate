#!/bin/bash

# This is a helper to update the bundles manifests file. This should be invoked
# via its make target (`make ocp-update-bundle-manifests`)

set -ex

source ./hack/ocp-bundle-common.sh

# remove old manifests & bundle metadata files
rm -rf ${BUNDLE_DIR}/manifests ${BUNDLE_DIR}/metadata

# generate bundle files from scratch
IMAGE_REPO=${IMAGE_REPO} \
HANDLER_IMAGE_NAME=${HANDLER_IMAGE_NAME} HANDLER_IMAGE_TAG=${HANDLER_IMAGE_TAG} HANDLER_NAMESPACE=${HANDLER_NAMESPACE} \
OPERATOR_IMAGE_NAME=${OPERATOR_IMAGE_NAME} OPERATOR_IMAGE_TAG=${OPERATOR_IMAGE_TAG} OPERATOR_NAMESPACE=${OPERATOR_NAMESPACE} \
PLUGIN_IMAGE_NAME=${PLUGIN_IMAGE_NAME} PLUGIN_IMAGE_TAG=${PLUGIN_IMAGE_TAG} PLUGIN_NAMESPACE=${PLUGIN_NAMESPACE} \
VERSION=${VERSION} CHANNELS=${CHANNEL},alpha DEFAULT_CHANNEL=${CHANNEL} \
BUNDLE_DIR=${BUNDLE_DIR} MANIFEST_BASES_DIR=${MANIFEST_BASES_DIR} make bundle

# add the cluster permissions to use the privileged security context constraint to the nmstate-operator SA in the CSV
$(yq4) --inplace eval '.spec.install.spec.clusterPermissions[] |= select(.rules[]) |= select(.serviceAccountName == "nmstate-operator").rules += {"apiGroups":["security.openshift.io"],"resources":["securitycontextconstraints"],"verbs":["use"],"resourceNames":["privileged"]}' ${BUNDLE_DIR}/manifests/kubernetes-nmstate-operator.clusterserviceversion.yaml

# remove unneeded owned CRDs in CSV / use only NMState v1 CRD
$(yq4) --inplace eval '.spec.customresourcedefinitions.owned |= [{"kind":"NMState","name":"nmstates.nmstate.io","version":"v1","description":"Represents an NMState deployment.","displayName":"NMState"}]' ${BUNDLE_DIR}/manifests/kubernetes-nmstate-operator.clusterserviceversion.yaml

# Add OpenShift required annotations (https://docs.engineering.redhat.com/pages/viewpage.action?spaceKey=CFC&title=Best_Practices)
$(yq4) --inplace eval '.metadata.annotations += {"features.operators.openshift.io/disconnected": "true"}' ${BUNDLE_DIR}/manifests/kubernetes-nmstate-operator.clusterserviceversion.yaml
$(yq4) --inplace eval '.metadata.annotations += {"features.operators.openshift.io/fips-compliant": "true"}' ${BUNDLE_DIR}/manifests/kubernetes-nmstate-operator.clusterserviceversion.yaml
$(yq4) --inplace eval '.metadata.annotations += {"features.operators.openshift.io/proxy-aware": "false"}' ${BUNDLE_DIR}/manifests/kubernetes-nmstate-operator.clusterserviceversion.yaml
$(yq4) --inplace eval '.metadata.annotations += {"features.operators.openshift.io/tls-profiles": "false"}' ${BUNDLE_DIR}/manifests/kubernetes-nmstate-operator.clusterserviceversion.yaml
$(yq4) --inplace eval '.metadata.annotations += {"features.operators.openshift.io/token-auth-aws": "false"}' ${BUNDLE_DIR}/manifests/kubernetes-nmstate-operator.clusterserviceversion.yaml
$(yq4) --inplace eval '.metadata.annotations += {"features.operators.openshift.io/token-auth-azure": "false"}' ${BUNDLE_DIR}/manifests/kubernetes-nmstate-operator.clusterserviceversion.yaml
$(yq4) --inplace eval '.metadata.annotations += {"features.operators.openshift.io/token-auth-gcp": "false"}' ${BUNDLE_DIR}/manifests/kubernetes-nmstate-operator.clusterserviceversion.yaml

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
      name: quay.io/openshift/origin-kubernetes-nmstate-operator:${VERSION_MAJOR_MINOR}
  - name: kubernetes-nmstate-handler
    from:
      kind: DockerImage
      name: quay.io/openshift/origin-kubernetes-nmstate-handler:${VERSION_MAJOR_MINOR}
  - name: nmstate-console-plugin-rhel8
    from:
      kind: DockerImage
      name: quay.io/openshift/origin-nmstate-console-plugin:${VERSION_MAJOR_MINOR}
EOF

# undo changes on "root" bundle.Dockerfile (gets updated by `make bundle`)
git checkout bundle.Dockerfile
