#!/bin/bash
set -e
node_name=${1}
node_ip=$(${KUBECTL} get node ${node_name} --no-headers -o wide | awk '{print $6}')
ssh core@${node_ip} -- ${@:3}
