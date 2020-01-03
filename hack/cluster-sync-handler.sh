#!/bin/bash -ex

script_dir=$(dirname "$(readlink -f "$0")")

${KUBECTL} apply -f deploy/crds/nmstate.io_nodenetworkstates_crd.yaml
${KUBECTL} apply -f deploy/crds/nmstate.io_nodenetworkconfigurationpolicies_crd.yaml
${KUBECTL} apply -f deploy/crds/nmstate.io_nodenetworkconfigurationenactments_crd.yaml
${KUBECTL} delete --ignore-not-found -f ${local_handler_manifest}

$script_dir/install-tls-secrets.sh \
		--namespace nmstate \
		--service nmstate-webhook \
		--secret nmstate-webhook-certs

# replace caBundle from webhook with real deal inspired by [1]
#
# [1] https://github.com/morvencao/kube-mutating-webhook-tutorial/blob/master/deployment/webhook-patch-ca-bundle.sh
CA_BUNDLE=$(${KUBECTL} get configmap -n kube-system extension-apiserver-authentication -o=jsonpath='{.data.client-ca-file}' | base64 | tr -d '\n')
sed -i "s/REPLACE_CA_BUNDLE/$CA_BUNDLE/g" ${local_handler_manifest}

# Set debug verbosity level for logs when using cluster-sync
sed "s#--v=production#--v=debug#" ${local_handler_manifest} | ${KUBECTL} create -f -

for i in {300..0}; do
    # We have to re-check desired number, sometimes takes some time to be filled in
    desiredNumberScheduled=$(${KUBECTL} get daemonset -n nmstate nmstate-handler -o=jsonpath='{.status.desiredNumberScheduled}')

    numberAvailable=$(${KUBECTL} get daemonset -n nmstate nmstate-handler -o=jsonpath='{.status.numberAvailable}')

    if [ "$desiredNumberScheduled" == "$numberAvailable" ]; then
        echo "nmstate-handler DS is ready"
        break
    fi

    if [ $i -eq 0 ]; then
        echo "nmstate-handler DS haven't turned ready within the given timeout"
    exit 1
    fi

    sleep 1
done
