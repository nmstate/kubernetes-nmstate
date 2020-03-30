#!/bin/bash -e

script_dir=$(dirname "$(readlink -f "$0")")
kubectl=$script_dir/../kubevirtci/cluster-up/kubectl.sh
number_of_nodes=$($kubectl get node  --no-headers |wc -l)

if [ $number_of_nodes -eq 1 ]; then
    label="node-role.kubernetes.io/master"
else
    label="!node-role.kubernetes.io/master"
fi

$kubectl label node -l $label node-role.kubernetes.io/worker=''
