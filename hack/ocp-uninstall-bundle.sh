
#!/bin/bash

set -ex

export NAMESPACE="openshift-nmstate"
export OPERATOR_NAMESPACE="${NAMESPACE}"

oc -n ${OPERATOR_NAMESPACE} delete ClusterServiceVersion $(oc -n ${OPERATOR_NAMESPACE} get Subscription kubernetes-nmstate-operator -ojsonpath='{.status.installedCSV}') || true
oc -n ${OPERATOR_NAMESPACE} delete Subscription kubernetes-nmstate-operator || true
oc -n ${OPERATOR_NAMESPACE} delete OperatorGroup openshift-kubernetes-nmstate-operator || true
oc -n openshift-marketplace delete CatalogSource kubernetes-nmstate-catalog || true
