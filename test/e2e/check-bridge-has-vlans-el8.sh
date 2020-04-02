#!/bin/bash -e

node=$1
connection=$2
vlan_min=$3
vlan_max=$4
vlans_out=/tmp/vlans.out

echo "Dumping bridge vlans to $vlans_out"
$(SSH) $node -- sudo bridge vlan show > $vlans_out

echo "Checking vlan range $vlan_min-$vlan_max at $connection"
for vlan in $(seq $vlan_min $vlan_max); do
    if ! cat $vlans_out|grep $connection -A 1 |grep " *$vlan *" > /dev/null; then
        echo "Vlan $vlan not found at $connection in node $node"
        return 1
    fi
done
