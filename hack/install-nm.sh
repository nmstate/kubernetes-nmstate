#!/bin/bash -e

# TODO: Iterate all the nodes
script_dir=$(dirname "$(readlink -f "$0")")
ssh=$script_dir/../kubevirtci/cluster-up/ssh.sh

if [[ "$KUBEVIRT_PROVIDER" =~ k8s ]]; then
    echo 'Install NetworkManager on the node'
    $ssh node01 -- sudo yum install -y NetworkManager
    $ssh node01 -- sudo systemctl daemon-reload
    $ssh node01 -- sudo systemctl restart NetworkManager

    echo 'Check NetworkManager is working fine'
    $ssh node01 -- nmcli device show > /dev/null
fi
