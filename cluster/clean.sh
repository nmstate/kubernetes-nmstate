#!/bin/bash -e

echo 'Cleaning up ...'

./cluster/kubectl.sh delete --ignore-not-found -f _out/manifests/namespace.yaml
./cluster/kubectl.sh delete --ignore-not-found -f _out/manifests/rbac.yaml
./cluster/kubectl.sh delete --ignore-not-found -f _out/manifests/state-crd.yaml
./cluster/kubectl.sh delete --ignore-not-found -f _out/manifests/configuration-policy-crd.yaml
./cluster/kubectl.sh delete --ignore-not-found -f _out/manifests/state-controller-ds.yaml
./cluster/kubectl.sh delete --ignore-not-found -f _out/manifests/state-client-pod.yaml

sleep 2

echo 'Done'
