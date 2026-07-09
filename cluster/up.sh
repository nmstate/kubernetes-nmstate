#!/bin/bash

set -ex

if [[ "$(uname -s)" == "Darwin" ]]; then
    echo "ERROR: make cluster-up is not supported on macOS." >&2
    echo "" >&2
    echo "kubevirtci requires x86_64 Linux and cannot run under arm64 emulation." >&2
    echo "Set up a Kubernetes cluster on a remote x86_64 Linux machine, then use:" >&2
    echo "" >&2
    echo "  export KUBEVIRT_PROVIDER=external" >&2
    echo "  export KUBECONFIG=/path/to/your/cluster/kubeconfig" >&2
    echo "  export DEV_IMAGE_REGISTRY=<your-registry>" >&2
    echo "  make cluster-sync" >&2
    echo "" >&2
    echo "See docs/content/developer-guide/106-macos-development.md for details." >&2
    exit 1
fi

source ./cluster/kubevirtci.sh
kubevirtci::install

$(kubevirtci::path)/cluster-up/up.sh

if [[ "$KUBEVIRT_PROVIDER" =~ ^(okd|ocp)-.*$$ ]]; then \
		while ! $(KUBECTL) get securitycontextconstraints; do sleep 1; done; \
fi

echo 'Upgrading NetworkManager and enabling and starting up openvswitch'
for node in $(./cluster/kubectl.sh get nodes --no-headers | awk '{print $1}'); do
    # Unload the unused emulated igb NIC. kubevirtci VMs ship an emulated
    # Intel 82576 (igb) NIC for SRIOV testing that kubernetes-nmstate does
    # not use (eth3). Reading its interface stats (igb_get_stats64 ->
    # igb_update_stats -> igb_rd32) traps into an emulated MMIO register
    # read that stalls for seconds under host load while holding the
    # adapter stats spinlock. Every stats reader then piles onto that lock:
    # kubelet/cadvisor via /proc/net/dev and nmstatectl/node_exporter via
    # RTM_GETLINK dumps, producing CPU soft lockups and 120s+ hangs that
    # fail the e2e handler suite.
    #
    # modprobe.d blacklisting cannot fix this across reboots: kubevirtci
    # boots each node from an immutable external kernel+initrd (qemu
    # -kernel/-initrd) and that initrd loads igb before the rootfs
    # modprobe.d is read, so igb reappears after every reboot performed by
    # the e2e suite. The node rootfs does persist, so install a systemd
    # oneshot unit there that unloads igb on every boot.
    ./cluster/cli.sh ssh ${node} -- sudo rm -f /etc/modules-load.d/igb.conf
    ./cluster/cli.sh ssh ${node} -- 'printf "[Unit]\nDescription=Unload unused emulated igb NIC (avoids stats-spinlock soft lockups)\n\n[Service]\nType=oneshot\nRemainAfterExit=yes\nExecStart=-/usr/sbin/modprobe -r igb\n\n[Install]\nWantedBy=multi-user.target\n" | sudo tee /etc/systemd/system/disable-igb.service > /dev/null'
    ./cluster/cli.sh ssh ${node} -- 'sudo systemctl enable --now disable-igb.service || true'
    if [[ "$NM_VERSION" == "latest" ]]; then
        echo "Installing NetworkManager from copr networkmanager/NetworkManager-main"
        ./cluster/cli.sh ssh ${node} -- sudo dnf install -y dnf-plugins-core
        ./cluster/cli.sh ssh ${node} -- sudo dnf copr enable -y networkmanager/NetworkManager-main
    fi
    ./cluster/cli.sh ssh ${node} -- sudo dnf upgrade -y NetworkManager --allowerasing
    ./cluster/cli.sh ssh ${node} -- sudo systemctl daemon-reload
    ./cluster/cli.sh ssh ${node} -- sudo systemctl enable openvswitch
    ./cluster/cli.sh ssh ${node} -- sudo systemctl restart openvswitch
    # Newer kubevirtci has dhclient installed so we should enforce not using it to
    # keep using the NM internal DHCP client as we always have
    ./cluster/cli.sh ssh ${node} -- sudo rm -f /etc/NetworkManager/conf.d/002-dhclient.conf
    ./cluster/cli.sh ssh ${node} -- sudo systemctl restart NetworkManager
    # Enable persistent journal so logs survive node reboots
    ./cluster/cli.sh ssh ${node} -- sudo mkdir -p /var/log/journal
    ./cluster/cli.sh ssh ${node} -- sudo systemctl restart systemd-journald
done

# Emulate a production LLDP-speaking switch on the cluster network so nodes
# receive LLDP neighbors on their NICs (consumed by the LLDP e2e tests)
./cluster/lldpd-switch.sh up
