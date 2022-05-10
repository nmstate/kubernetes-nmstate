#!/bin/bash -xe 

for node in $(./cluster/kubectl.sh get nodes --no-headers | awk '{print $1}'); do
    ./cluster/ssh.sh $node -- sudo crictl rmi ghcr.io/k8snetworkplumbingwg/whereabouts:latest-amd64 || true
done

whereabouts_url=https://raw.githubusercontent.com/k8snetworkplumbingwg/whereabouts/master/doc/crds

./cluster/kubectl.sh apply \
    -f $whereabouts_url/whereabouts.cni.cncf.io_ippools.yaml \
    -f $whereabouts_url/whereabouts.cni.cncf.io_overlappingrangeipreservations.yaml \
