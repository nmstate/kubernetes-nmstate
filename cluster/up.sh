#!/bin/bash

set -ex

source ./cluster/kubevirtci.sh
kubevirtci::install

$(kubevirtci::path)/cluster-up/up.sh

if [[ "$KUBEVIRT_PROVIDER" =~ ^(okd|ocp)-.*$$ ]]; then \
		while ! $(KUBECTL) get securitycontextconstraints; do sleep 1; done; \
fi

if [[ "$KUBEVIRT_PROVIDER" =~ k8s- ]]; then
    echo 'Install Open vSwitch on nodes'
    for node in $(./cluster/kubectl.sh get nodes --no-headers | awk '{print $1}'); do
        ./cluster/cli.sh ssh ${node} sudo -- yum install -y http://cbs.centos.org/kojifiles/packages/openvswitch/2.9.2/1.el7/x86_64/openvswitch-2.9.2-1.el7.x86_64.rpm http://cbs.centos.org/kojifiles/packages/openvswitch/2.9.2/1.el7/x86_64/openvswitch-devel-2.9.2-1.el7.x86_64.rpm http://cbs.centos.org/kojifiles/packages/dpdk/17.11/3.el7/x86_64/dpdk-17.11-3.el7.x86_64.rpm
        ./cluster/cli.sh ssh ${node} sudo -- systemctl daemon-reload
        ./cluster/cli.sh ssh ${node} sudo -- systemctl restart openvswitch
    done

    echo 'Install NetworkManager on nodes'
    for node in $(./cluster/kubectl.sh get nodes --no-headers | awk '{print $1}'); do
        ./cluster/cli.sh ssh ${node} sudo -- yum install -y yum-plugin-copr
        ./cluster/cli.sh ssh ${node} sudo -- yum copr enable -y networkmanager/NetworkManager-1.22
        ./cluster/cli.sh ssh ${node} sudo -- yum install -y NetworkManager NetworkManager-ovs
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

