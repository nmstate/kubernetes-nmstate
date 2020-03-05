#!/bin/bash

set -ex

function getDesiredNumberScheduled {
    echo $(./cluster/kubectl.sh get daemonset -n nmstate $1 -o=jsonpath='{.status.desiredNumberScheduled}')
}

function getNumberAvailable {
    numberAvailable=$(./cluster/kubectl.sh get daemonset -n nmstate $1 -o=jsonpath='{.status.numberAvailable}')
    echo ${numberAvailable:-0}
}

function consistently {
    cmd=$@
    retries=3
    interval=1
    cnt=1
    while [[ $cnt -le $retries ]]; do
        $cmd
        sleep $interval
        cnt=$(($cnt + 1))
    done
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

# Cleanup previous deployment, if there is any
make cluster-clean

# Fetch registry port that can be used to upload images to the local kubevirtci cluster
registry_port=$(./cluster/cli.sh ports registry | tr -d '\r')
if [[ "${KUBEVIRT_PROVIDER}" =~ ^(okd|ocp)-.*$ ]]; then \
		registry=localhost:$(./cluster/cli.sh ports --container-name=cluster registry | tr -d '\r')
else
    registry=localhost:$(./cluster/cli.sh ports registry | tr -d '\r')
fi

# Build new handler image from local sources and push it to the kubevirtci cluster
IMAGE_REGISTRY=${registry} make push-handler

# Deploy all needed manifests
./cluster/kubectl.sh apply -f deploy/namespace.yaml
./cluster/kubectl.sh apply -f deploy/service_account.yaml
./cluster/kubectl.sh apply -f deploy/role.yaml
./cluster/kubectl.sh apply -f deploy/role_binding.yaml
./cluster/kubectl.sh apply -f deploy/crds/nmstate.io_nodenetworkstates_crd.yaml
./cluster/kubectl.sh apply -f deploy/crds/nmstate.io_nodenetworkconfigurationpolicies_crd.yaml
./cluster/kubectl.sh apply -f deploy/crds/nmstate.io_nodenetworkconfigurationenactments_crd.yaml
if [[ "$KUBEVIRT_PROVIDER" =~ ^(okd|ocp)-.*$ ]]; then
		./cluster/kubectl.sh apply -f deploy/openshift/
fi
sed \
    -e "s#--v=production#--v=debug#" \
    -e "s#REPLACE_IMAGE#registry:5000/nmstate/kubernetes-nmstate-handler#" \
    deploy/operator.yaml | ./cluster/kubectl.sh create -f -

# Wait until the handler becomes ready on all nodes
for i in {300..0}; do
    # We have to re-check desired number, sometimes takes some time to be filled in
    if consistently isOk nmstate-handler && consistently isOk nmstate-handler-worker; then
       break
    fi

    if [ $i -eq 0 ]; then
        echo "nmstate-handler or nmstate-handler-worker DS haven't turned ready within the given timeout"
    exit 1
    fi

    sleep 1
done
