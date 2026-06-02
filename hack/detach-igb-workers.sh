#!/bin/bash

set -euo pipefail

workers="$(./cluster/kubectl.sh get nodes -l node-role.kubernetes.io/worker -o jsonpath='{.items[*].metadata.name}')"
if [ -z "${workers// }" ]; then
    echo "No worker nodes found via label node-role.kubernetes.io/worker"
    exit 1
fi

read -r -d '' remote_script <<'EOF' || true
set -euo pipefail

matches="$(
for ifpath in /sys/class/net/*; do
    dev=${ifpath##*/}
    [ "$dev" = "lo" ] && continue
    [ -L "$ifpath/device/driver" ] || continue
    [ "$(basename "$(readlink -f "$ifpath/device/driver")")" = "igb" ] || continue
    pci=$(basename "$(readlink -f "$ifpath/device")")
    lspci -nns "$pci" | grep -qi "\[8086:10c9\]" || continue
    echo "$dev $pci"
done
)"

count=$(printf "%s\n" "$matches" | sed "/^$/d" | wc -l)
if [ "$count" -eq 0 ]; then
    echo "SKIP: no bound igb [8086:10c9] device found"
    exit 0
fi
if [ "$count" -ne 1 ]; then
    echo "ERROR: multiple bound igb [8086:10c9] devices found"
    printf "%s\n" "$matches" | sed "/^$/d" | sed "s/^/  /"
    exit 1
fi

read -r dev pci <<<"$(printf "%s\n" "$matches" | sed "/^$/d")"
echo "Target: $dev ($pci)"

nmcli device set "$dev" managed no || true
ip link set dev "$dev" down || true

echo "$pci" >"/sys/bus/pci/devices/$pci/driver/unbind"

if [ -L "/sys/bus/pci/devices/$pci/driver" ]; then
    echo "ERROR: still bound"
    exit 1
fi

if ip -o link show "$dev" >/dev/null 2>&1; then
    echo "ERROR: $dev still exists"
    exit 1
fi

if nmcli -t -f DEVICE device status | grep -qx "$dev"; then
    echo "ERROR: NetworkManager still lists $dev"
    exit 1
fi

if lspci -nnk -s "$pci" 2>/dev/null | grep -q 'Kernel driver in use:'; then
    echo "ERROR: lspci still reports a bound kernel driver"
    exit 1
fi

echo "OK: detached $dev ($pci)"
EOF

remote_quoted="$(printf '%q' "$remote_script")"

for node in $workers; do
    echo "===== ${node} ====="
    ./cluster/ssh.sh "$node" "sudo bash -lc $remote_quoted"
done
