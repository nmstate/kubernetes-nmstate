#!/bin/bash
node_name=${1}
node_ip=$(oc get node ${node_name} --no-headers -o wide | awk '{print $6}')
ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null core@${node_ip} -- ${@:3}
