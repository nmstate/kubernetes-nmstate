#!/bin/bash

set -ex

source ./cluster/kubevirtci.sh
kubevirtci::install

$(kubevirtci::path)/cluster-up/up.sh

if [[ "$KUBEVIRT_PROVIDER" =~ ^(okd|ocp)-.*$$ ]]; then \
		while ! $(KUBECTL) get securitycontextconstraints; do sleep 1; done; \
fi

if [[ "$KUBEVIRT_PROVIDER" =~ k8s- ]]; then
    echo 'Install NetworkManager on nodes'
    for node in $(./cluster/kubectl.sh get nodes --no-headers | awk '{print $1}'); do
        ./cluster/cli.sh ssh ${node} sudo -- yum install -y yum-plugin-copr
        ./cluster/cli.sh ssh ${node} sudo -- yum copr enable -y networkmanager/$NM_COPR_REPO
        ./cluster/cli.sh ssh ${node} sudo -- yum install -y NetworkManager
        ./cluster/cli.sh ssh ${node} sudo -- systemctl daemon-reload
        ./cluster/cli.sh ssh ${node} sudo -- systemctl restart NetworkManager
        echo "Check NetworkManager is working fine on node $node"
        ./cluster/cli.sh ssh ${node} -- nmcli device show > /dev/null
    done
fi

for node in $(./cluster/kubectl.sh get nodes --no-headers | awk '{print $1}'); do
    for nic in $FIRST_SECONDARY_NIC $SECOND_SECONDARY_NIC; do
	      uuid=$(./cluster/cli.sh ssh $node -- nmcli --fields=device,uuid  c show  |grep $nic|awk '{print $2}')
	      if [ ! -z "$uuid" ]; then
        	  echo "$node: Flushing nic $nic"
        	  ./cluster/cli.sh ssh $node -- sudo nmcli con del $uuid
	      fi
    done
done

