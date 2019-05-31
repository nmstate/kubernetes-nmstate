#!/bin/bash -e

registry_port=$(./cluster/cli.sh ports registry | tr -d '\r')
registry=localhost:$registry_port

IMAGE_REGISTRY=${registry} make push-manager

./cluster/cli.sh ssh node01 'sudo docker pull registry:5000/nmstate/kubernetes-nmstate-manager'
# Temporary until image is updated with provisioner that sets this field
# This field is required by buildah tool
./cluster/cli.sh ssh node01 'sudo sysctl -w user.max_user_namespaces=1024'
