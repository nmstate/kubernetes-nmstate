#!/bin/bash -ex

${KUBECTL} apply -f deploy/crds/nmstate.io_nodenetworkstates_crd.yaml
${KUBECTL} apply -f deploy/crds/nmstate.io_nodenetworkconfigurationpolicies_crd.yaml
${KUBECTL} delete --ignore-not-found -f ${local_handler_manifest}
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
