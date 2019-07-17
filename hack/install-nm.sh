#!/bin/bash -xe

# TODO: Iterate all the nodes

pwd

ssh=kubevirtci/cluster-up/ssh.sh
kubectl=kubevirtci/cluster-up/kubectl.sh

echo 'Install Open vSwitch on nodes'
if [[ "$KUBEVIRT_PROVIDER" =~  k8s- ]]; then
    for i in $(seq 1 ${KUBEVIRT_NUM_NODES}); do
        node_name="node$(printf "%02d" ${i})"
        $ssh $node_name -- sudo yum install -y http://cbs.centos.org/kojifiles/packages/openvswitch/2.9.2/1.el7/x86_64/openvswitch-2.9.2-1.el7.x86_64.rpm http://cbs.centos.org/kojifiles/packages/openvswitch/2.9.2/1.el7/x86_64/openvswitch-devel-2.9.2-1.el7.x86_64.rpm http://cbs.centos.org/kojifiles/packages/dpdk/17.11/3.el7/x86_64/dpdk-17.11-3.el7.x86_64.rpm
        $ssh $node_name -- sudo systemctl daemon-reload
        $ssh $node_name -- sudo systemctl restart openvswitch
    done
elif [[ "$KUBEVIRT_PROVIDER" =~ os- ]]; then
    $kubectl create -f deploy/openshift/ovs-vsctl.yaml
    until [[ $($kubectl -n kube-system get daemonsets | grep ovs-vsctl-amd64 | awk '{ if ($3 == $4) print "0"; else print "1"}') -eq "0" ]]; do
        sleep 1
    done
fi

echo 'Install NetworkManager on nodes'
for i in $(seq 1 ${KUBEVIRT_NUM_NODES}); do
    node_name="node$(printf "%02d" ${i})"
    $ssh $node_name sudo -- yum install -y NetworkManager NetworkManager-ovs
    $ssh $node_name sudo -- systemctl daemon-reload
    $ssh $node_name sudo -- systemctl restart NetworkManager
    echo 'Check NetworkManager is working fine'
    $ssh $node_name -- nmcli device show > /dev/null
done

