#!/bin/bash -e


function install_ovs_on_node() {
    node=$1
    # FIXME: We have some issue with centos7 and resolving cbs.centos.org
    # we may need to remove this in the future
    $SSH $node -- sudo yum install -y bind-utils
    $SSH $node -- sudo yum install -y \
        https://cbs.centos.org/kojifiles/packages/openvswitch/2.9.2/1.el7/x86_64/openvswitch-2.9.2-1.el7.x86_64.rpm \
        https://cbs.centos.org/kojifiles/packages/openvswitch/2.9.2/1.el7/x86_64/openvswitch-devel-2.9.2-1.el7.x86_64.rpm \
        https://cbs.centos.org/kojifiles/packages/dpdk/17.11/3.el7/x86_64/dpdk-17.11-3.el7.x86_64.rpm
    $SSH $node -- sudo systemctl daemon-reload
    $SSH $node -- sudo systemctl restart openvswitch
}

# we currently skip ovs for non k8s providers.
if [[ "$KUBEVIRT_PROVIDER" =~  k8s- ]]; then
    echo 'Installing Open vSwitch on nodes'

    for node in $($KUBECTL get nodes --no-headers | awk '{print $1}'); do
        install_ovs_on_node "$node"
    done
fi
