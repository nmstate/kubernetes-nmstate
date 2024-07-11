#!/bin/bash
node_name=${1}
command=${@:3}

execute=(oc -n default debug node/"${node_name}" -- chroot /host ${command})
"${execute[@]}"
