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

function isOpenshift {
    $kubectl get co openshift-apiserver
}

function digest {
    local registry=${1}
    local image=${2}
    skopeo inspect --tls-verify=false docker://${registry}/nmstate/kubernetes-nmstate-${image}:latest |jq -r '.Digest'
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

    # KIND providers handle registry differently than k8s-* providers
    if [[ "${KUBEVIRT_PROVIDER}" =~ ^kind-.*$ ]]; then
        # For KIND, the registry port is determined by the provider config
        # Default is 5000 for the default cluster name, 5001 for alternate names
        if [[ "${CLUSTER_NAME:-kind-${KUBEVIRT_PROVIDER#kind-}}" == "kind-${KUBEVIRT_PROVIDER#kind-}" ]]; then
            registry_port=5000
        else
            registry_port=5001
        fi
        registry=localhost:${registry_port}
    else
        # Fetch registry port that can be used to upload images to the local kubevirtci cluster
        registry_port=$(./cluster/cli.sh ports registry | tr -d '\r')
        if [[ "${KUBEVIRT_PROVIDER}" =~ ^(okd|ocp)-.*$ ]]; then \
                registry=localhost:$(./cluster/cli.sh ports --container-name=cluster registry | tr -d '\r')
        else
            registry=localhost:$(./cluster/cli.sh ports registry | tr -d '\r')
        fi
    fi

    # Build new handler and operator image from local sources and push it to the kubevirtci cluster
    IMAGE_REGISTRY=${registry} make push

    # Generate the manifests potinting to the sha256 digest of the pushed images and the local registry
    export OPERATOR_IMAGE_FULL_NAME=nmstate/kubernetes-nmstate-operator@$(digest $registry operator)
    export HANDLER_IMAGE_FULL_NAME=nmstate/kubernetes-nmstate-handler@$(digest $registry handler)
    export IMAGE_REGISTRY=registry:5000
    make manifests
}
