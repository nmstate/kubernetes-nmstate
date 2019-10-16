#!/bin/bash -e

# TODO: Iterate all the nodes
script_dir=$(dirname "$(readlink -f "$0")")
ssh=$script_dir/../kubevirtci/cluster-up/ssh.sh

# TODO okd4.1
if [[ "$KUBEVIRT_PROVIDER" =~ k8s ]]; then
    for n in $(seq 1 $KUBEVIRT_NUM_SECONDARY_NICS); do
        echo 'Flushing nic $n'
        $ssh node01 -- sudo nmcli con del "\"Wired connection $n\""
    done
fi
