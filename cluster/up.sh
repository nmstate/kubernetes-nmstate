#!/bin/bash

set -ex

source ./cluster/kubevirtci.sh
kubevirtci::install

$(kubevirtci::path)/cluster-up/up.sh

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
    for nic in $(nmcli --fields NAME,UUID -t con show | grep 'Wired Connection' | awk -F : '{print $2}'); do
        nmcli con modify $nic match.interface-name "!$FIRST_SECONDARY_NIC, !$SECOND_SECONDARY_NIC";
    done
done

echo 'Upgrading NetworkManager and enabling and starting up openvswitch'
for node in $(./cluster/kubectl.sh get nodes --no-headers | awk '{print $1}'); do
    ./cluster/cli.sh ssh ${node} -- sudo dnf upgrade -y NetworkManager
    ./cluster/cli.sh ssh ${node} -- sudo systemctl daemon-reload
    ./cluster/cli.sh ssh ${node} -- sudo systemctl enable openvswitch
    ./cluster/cli.sh ssh ${node} -- sudo systemctl restart openvswitch
    ./cluster/cli.sh ssh ${node} -- sudo systemctl restart NetworkManager
done
