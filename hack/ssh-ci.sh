#!/bin/bash
node_name=${1}
node_ip=$(oc get node ${node_name} --no-headers -o wide | awk '{print $6}')
IP="$(cat ${SHARED_DIR}/server-ip)"
SSHOPTS="-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null"
       
ssh ${SSHOPTS} -i ${CLUSTER_PROFILE_DIR}/packet-ssh-key root@${IP} "ssh ${SSHOPTS} core@${node_ip} -- ${@:3}"
