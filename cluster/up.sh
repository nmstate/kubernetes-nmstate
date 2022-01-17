#!/bin/bash

set -ex

source ./cluster/kubevirtci.sh
kubevirtci::install

$(kubevirtci::path)/cluster-up/up.sh

nm_version=1.36.0-0.4

if [[ "$KUBEVIRT_PROVIDER" =~ ^(okd|ocp)-.*$$ ]]; then \
		while ! $(KUBECTL) get securitycontextconstraints; do sleep 1; done; \
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

echo 'Installing Open vSwitch and NetworkManager 1.34 on nodes'
for node in $(./cluster/kubectl.sh get nodes --no-headers | awk '{print $1}'); do
    ./cluster/cli.sh ssh ${node} -- sudo dnf config-manager --set-enabled powertools
    ./cluster/cli.sh ssh ${node} -- sudo dnf install -y epel-release centos-release-openstack-wallaby
    ./cluster/cli.sh ssh ${node} -- sudo dnf install -y openvswitch libibverbs NetworkManager-ovs-$nm_version NetworkManager-$nm_version
    ./cluster/cli.sh ssh ${node} -- sudo systemctl daemon-reload
    ./cluster/cli.sh ssh ${node} -- sudo systemctl enable openvswitch
    ./cluster/cli.sh ssh ${node} -- sudo systemctl restart openvswitch
    ./cluster/cli.sh ssh ${node} -- sudo systemctl restart NetworkManager
done
