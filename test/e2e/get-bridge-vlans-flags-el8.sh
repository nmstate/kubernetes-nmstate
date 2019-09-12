#!/bin/bash -e

node=$1
connection=$2
vlan=$3
vlans_out=/tmp/vlans.out

./kubevirtci/cluster-up/ssh.sh $node -- sudo bridge vlan show > $vlans_out

tags=$(grep $connection$ -A 1 $vlans_out |sed "s/\t/\n/g" | grep " $vlan " | sed "s/ $vlan *//")
echo -n $tags
