function eventually {
    timeout=$(( $KUBEVIRT_NUM_NODES * 30 ))
    interval=5
    cmd=$@
    echo "Checking eventually $cmd"
    while ! $cmd; do
        if [ $timeout -le 0 ]; then
            return 1
        fi
        sleep $interval
        timeout=$(( $timeout - $interval ))
    done
}

function consistently {
    timeout=15
    interval=5
    cmd=$@
    echo "Checking consistently $cmd"
    while $cmd; do
        if [ $timeout -le 0 ]; then
            return 0
        fi
        sleep $interval
        timeout=$(( $timeout - $interval ))
    done
    return 1
}

function isExternal {
    [[ "${KUBEVIRT_PROVIDER}" == external ]]
}

function isDeploymentOk {
    $kubectl wait deployment -n $1 -l $2 --for condition=Available --timeout=15s
}

function isDaemonSetOk {
    desiredNumberScheduled=$($kubectl get daemonset -n $1 -l $2 -o=jsonpath='{..status.desiredNumberScheduled}')

    numberAvailable=$($kubectl get daemonset -n $1 -l $2 -o=jsonpath='{..status.numberAvailable}')

    # There is no numberAvailable yet, return error so we don't end up with
    # false possitive after 0==0
    if [ "$numberAvailable" == "" ]; then
        return 1
    fi

    [ "$desiredNumberScheduled" == "$numberAvailable" ]
}

function isOpenshift {
    $kubectl get co openshift-apiserver
}

function push() {
    if isExternal; then
        if [[ ! -v DEV_IMAGE_REGISTRY ]]; then
            echo "Missing DEV_IMAGE_REGISTRY variable"
            return 1
        fi
        make IMAGE_REGISTRY=$DEV_IMAGE_REGISTRY manifests push
        return 0
    fi
    # Fetch registry port that can be used to upload images to the local kubevirtci cluster
    registry_port=$(./cluster/cli.sh ports registry | tr -d '\r')
    if [[ "${KUBEVIRT_PROVIDER}" =~ ^(okd|ocp)-.*$ ]]; then \
            registry=localhost:$(./cluster/cli.sh ports --container-name=cluster registry | tr -d '\r')
    else
        registry=localhost:$(./cluster/cli.sh ports registry | tr -d '\r')
    fi

    # Build new handler and operator image from local sources and push it to the kubevirtci cluster
    IMAGE_REGISTRY=${registry} make push

    # Also generate the manifests pointing to the local registry
    IMAGE_REGISTRY=registry:5000 make manifests
}
