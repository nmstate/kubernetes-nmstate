#!/bin/bash -e

# TODO: Iterate all the nodes

node01_id=$(docker ps |grep node01 |awk '{print $1}')

echo 'Install NetworkManager on the node'
docker exec $node01_id ssh.sh sudo yum install -y NetworkManager
docker exec $node01_id ssh.sh sudo systemctl daemon-reload
docker exec $node01_id ssh.sh sudo systemctl restart NetworkManager

echo 'Check NetworkManager is working fine'
docker exec $node01_id ssh.sh nmcli device show > /dev/null
