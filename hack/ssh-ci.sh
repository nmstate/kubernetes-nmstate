#!/bin/bash
node_name=${1}
node_ip=$(oc get node ${node_name} --no-headers -o wide | awk '{print $6}')
IP="$(cat ${SHARED_DIR}/server-ip)"
PORT="$(cat ${SHARED_DIR}/server-sshport)"
SSHOPTS="-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null"

# A hack to avoid "Could not create directory '/.ssh'" error because inside container
# home is directly at root
export HOME=/root
SSHKEYINSIDEPACKET="/root/.ssh/id_rsa"

ssh ${SSHOPTS} -i ${CLUSTER_PROFILE_DIR}/packet-ssh-key root@${IP} -p ${PORT} "ssh ${SSHOPTS} -i ${SSHKEYINSIDEPACKET} core@${node_ip} -- ${@:3}"
