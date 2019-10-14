#!/bin/bash -e

ssh=kubevirtci/cluster-up/ssh.sh
kubectl=kubevirtci/cluster-up/kubectl.sh

function install_nm_on_node() {
    node=$1
    $ssh $node sudo -- yum install -y NetworkManager NetworkManager-ovs
    $ssh $node sudo -- systemctl daemon-reload
    $ssh $node sudo -- systemctl restart NetworkManager
    echo "Check NetworkManager is working fine on node $node"
    $ssh $node -- nmcli device show > /dev/null
}

echo 'Install NetworkManager on nodes'
for node in $($kubectl get nodes --no-headers | awk '{print $1}'); do
    install_nm_on_node "$node"
done
