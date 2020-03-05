#!/bin/bash

set -xe

node=$1
connection=$2
vlan=$3
vlans_out=/tmp/vlans.out

./cluster/cli.sh ssh $node -- sudo bridge vlan show > $vlans_out

# Remove CR from output
sed -i 's/\r$//' $vlans_out

tags=$(grep $connection$ -A 1 $vlans_out |sed "s/\t/\n/g" | grep " $vlan " | sed "s/ $vlan *//")
echo -n $tags
