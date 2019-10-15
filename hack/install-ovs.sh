#!/bin/bash -e

ssh=kubevirtci/cluster-up/ssh.sh
kubectl=kubevirtci/cluster-up/kubectl.sh


function install_ovs_on_node() {
    node=$1
    $ssh $node -- sudo yum install -y http://cbs.centos.org/kojifiles/packages/openvswitch/2.9.2/1.el7/x86_64/openvswitch-2.9.2-1.el7.x86_64.rpm http://cbs.centos.org/kojifiles/packages/openvswitch/2.9.2/1.el7/x86_64/openvswitch-devel-2.9.2-1.el7.x86_64.rpm http://cbs.centos.org/kojifiles/packages/dpdk/17.11/3.el7/x86_64/dpdk-17.11-3.el7.x86_64.rpm
    $ssh $node -- sudo systemctl daemon-reload
    $ssh $node -- sudo systemctl restart openvswitch
}

# we currently skip ovs for non k8s providers.
if [[ "$KUBEVIRT_PROVIDER" =~  k8s- ]]; then
    echo 'Installing Open vSwitch on nodes'

    for node in $($kubectl get nodes --no-headers | awk '{print $1}'); do
        install_ovs_on_node "$node"
    done
fi
