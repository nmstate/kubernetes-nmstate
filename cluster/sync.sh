#!/bin/bash -e

registry_port=$(./cluster/cli.sh ports registry | tr -d '\r')
registry=localhost:$registry_port

IMAGE_REGISTRY=${registry} make docker docker-push
MANIFESTS_DESTINATION='_out/manifests' IMAGE_REGISTRY='registry:5000' make manifests

for i in $(seq 1 ${KUBEVIRT_NUM_NODES}); do
    ./cluster/cli.sh ssh "node$(printf "%02d" ${i})" 'sudo docker pull registry:5000/kubernetes-nmstate-state-handler'
    ./cluster/cli.sh ssh "node$(printf "%02d" ${i})" 'sudo docker pull registry:5000/kubernetes-nmstate-configuration-policy-handler'
    # Temporary until image is updated with provisioner that sets this field
    # This field is required by buildah tool
    ./cluster/cli.sh ssh "node$(printf "%02d" ${i})" 'sudo sysctl -w user.max_user_namespaces=1024'
done

./cluster/kubectl.sh apply -f _out/manifests/namespace.yaml
./cluster/kubectl.sh apply -f _out/manifests/rbac.yaml
./cluster/kubectl.sh apply -f _out/manifests/state-crd.yaml
./cluster/kubectl.sh apply -f _out/manifests/configuration-policy-crd.yaml
if [[ $KUBEVIRT_PROVIDER =~ ^os-.*$ ]]; then
    ./cluster/kubectl.sh apply -f _out/manifests/openshift-scc.yaml
fi
