#!/bin/bash -e
#
#    Flush ip address at secondary nics
#
#    Our e2e tests expect secondary nics without IP addresses also
#    the ip addresses assigned by kubevirtci dnsmasq has the same CIDR
#    as the primary nic, this make impossible to test default-bridge
#    test propertly.
#

script_dir=$(dirname "$(readlink -f "$0")")
cluster_dir=$script_dir/../cluster
ssh=$cluster_dir/cli.sh ssh
kubectl=$cluster_dir/kubectl.sh

for node in $($kubectl get nodes --no-headers | awk '{print $1}'); do
    for nic in $FIRST_SECONDARY_NIC $SECOND_SECONDARY_NIC; do
	uuid=$($ssh $node -- nmcli --fields=device,uuid  c show  |grep $nic|awk '{print $2}')
	if [ ! -z "$uuid" ]; then
        	echo "$node: Flushing nic $nic"
        	$ssh $node -- sudo nmcli con del $uuid
	fi
    done
done
