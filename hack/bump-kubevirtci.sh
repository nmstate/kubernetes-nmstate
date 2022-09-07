#!/bin/bash -e

KUBEVIRTCI_TAG=$(curl -L -Ss https://storage.googleapis.com/kubevirt-prow/release/kubevirt/kubevirtci/latest)
[[ ${#KUBEVIRTCI_TAG} != "18" ]] && echo "error getting KUBEVIRTCI_TAG" && exit 1

sed -i "s/export KUBEVIRTCI_TAG=.*/export KUBEVIRTCI_TAG=${KUBEVIRTCI_TAG}/g" cluster/kubevirtci.sh

git --no-pager diff cluster/kubevirtci.sh | grep KUBEVIRTCI_TAG || true
