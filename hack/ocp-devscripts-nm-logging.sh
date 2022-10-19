#!/bin/bash

# This script captures the logs of network manager, network manager dispatcher
# and the entire journal of the cluster nodes created by dev-scripts. 
# This script may not run outside the VM host!

set -e

pids=()
nodes=()

if ! sudo virsh list | grep ostest >/dev/null; then
    echo "This script can only be run on the VM host as it needs direct ssh access to the VMs"
    exit 1
fi

echo "Getting cluster nodes..."
for node in $(oc get no -ojsonpath='{.items[*].metadata.name}'); do 
	nodes+=( ${node%%.*} )
done

read -p "Do you want me to cleanup ~/.ssh/known_hosts file and add the new keys of the cluster nodes? (y/n) " yn
case $yn in
	[yY] ) 
		for no in ${nodes[@]}; do sed -i "/^${no}/d" ~/.ssh/known_hosts; done
		for no in ${nodes[@]}; do ssh-keyscan -t ecdsa ${no} >> ~/.ssh/known_hosts; done
		;;
	[nN] ) 
		echo "Not cleaning up"
		;;
	* ) 
		echo "Invalid response"
		exit 1;;
esac

read -p "Do you want me to enable trace logging in NetworkManager? (y/n) " yn
case $yn in
	[yY] ) 
		echo "Enable trace logging in NetworkManager..."
		for node in ${nodes[@]}; do
			ssh core@${node} sudo sed -i 's/^\#*level=.*$/level=TRACE/g' /etc/NetworkManager/NetworkManager.conf
			ssh core@${node} sudo systemctl restart NetworkManager
            echo "Trace logging enabled on ${node}"
		done
		;;
	[nN] ) 
		echo "No trace logging enabled"
		;;
	* ) 
		echo "Invalid response"
		exit 1;;
esac

read -p "Press any key to start logging... " -n1 -s
echo
echo "Starting logging jobs..."
date

mkdir -p out/
for node in ${nodes[@]}; do
	echo "starting on node ${node} in background..."
	ssh core@${node} sudo journalctl --since now -xefu NetworkManager > out/networkmanager_${node}.log &
	pids+=( "$!" )
	
	ssh core@${node} sudo journalctl --since now -xefu NetworkManager-dispatcher > out/networkmanager-dispatcher_${node}.log &
    pids+=( "$!" )

	ssh core@${node} sudo journalctl --since now -xef > out/journal_${node}.log &
    pids+=( "$!" )
done

read -p "Press any key to stop logging... " -n1 -s
echo

for pid in ${pids[@]}; do
	kill $pid
done
date

echo "Done"
