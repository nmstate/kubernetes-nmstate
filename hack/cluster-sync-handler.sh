#!/bin/bash -ex

${KUBECTL} apply -f deploy/crds/nmstate.io_nodenetworkstates_crd.yaml
${KUBECTL} apply -f deploy/crds/nmstate.io_nodenetworkconfigurationpolicies_crd.yaml
${KUBECTL} apply -f deploy/crds/nmstate.io_nodenetworkconfigurationenactments_crd.yaml
${KUBECTL} delete --ignore-not-found -f ${local_handler_manifest}

# Set debug verbosity level for logs when using cluster-sync
sed "s#--v=production#--v=debug#" ${local_handler_manifest} | ${KUBECTL} create -f -

function getDesiredNumberScheduled {
        echo $(${KUBECTL} get daemonset -n nmstate $1 -o=jsonpath='{.status.desiredNumberScheduled}')
}

function getNumberAvailable {
        echo $(${KUBECTL} get daemonset -n nmstate $1 -o=jsonpath='{.status.numberAvailable}')
}

function isOk {
        desiredNumberScheduled=$(getDesiredNumberScheduled $1)
        numberAvailable=$(getNumberAvailable $1)


        if [ "$desiredNumberScheduled" == "$numberAvailable" ]; then
          echo "$1 DS is ready"
          return 0
        else
          return 1
        fi
}

for i in {300..0}; do
    # We have to re-check desired number, sometimes takes some time to be filled in
    if isOk nmstate-handler && isOk nmstate-handler-worker; then
       break
    fi

    if [ $i -eq 0 ]; then
        echo "nmstate-handler or nmstate-handler-worker DS haven't turned ready within the given timeout"
    exit 1
    fi


    sleep 1
done
