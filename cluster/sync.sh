#!/bin/bash -e

registry_port=$(./cluster/cli.sh ports registry | tr -d '\r')
registry=localhost:$registry_port

IMAGE_REGISTRY=${registry} make build-manager push-manager

./cluster/cli.sh ssh node01 'sudo docker pull registry:5000/kubernetes-nmstate-manager'
# Temporary until image is updated with provisioner that sets this field
# This field is required by buildah tool
./cluster/cli.sh ssh node01 'sudo sysctl -w user.max_user_namespaces=1024'

./cluster/kubectl.sh apply -f deploy/service_account.yaml
./cluster/kubectl.sh apply -f deploy/role.yaml
./cluster/kubectl.sh apply -f deploy/role_binding.yaml
./cluster/kubectl.sh apply -f deploy/crds/nmstate_v1_nodenetworkstate_crd.yaml
sed 's#REPLACE_IMAGE#registry:5000/kubernetes-nmstate-manager#' deploy/operator.yaml | ./cluster/kubectl.sh apply -f -
