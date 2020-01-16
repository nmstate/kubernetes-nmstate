#!/bin/bash -e

# TODO: Iterate all the nodes
script_dir=$(dirname "$(readlink -f "$0")")
ssh=$script_dir/../kubevirtci/cluster-up/ssh.sh

# TODO okd/ocp
if [[ "$KUBEVIRT_PROVIDER" =~ k8s ]]; then
    for node in $($KUBECTL get nodes --no-headers | awk '{print $1}'); do
        for n in $(seq 1 $KUBEVIRT_NUM_SECONDARY_NICS); do
            echo "$node: Flushing nic $n"
            $ssh $node -- sudo nmcli con del "\"Wired connection $n\""
        done
        echo "$node: restoring resolv.conf config"
        $ssh $node -- sudo dhclient -r eth0
        $ssh $node -- sudo dhclient eth0
    done
fi
