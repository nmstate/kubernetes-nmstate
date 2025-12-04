#!/bin/bash

set -ex

source ./cluster/kubevirtci.sh
kubevirtci::install

$(kubevirtci::path)/cluster-up/up.sh

if [[ "$KUBEVIRT_PROVIDER" =~ ^(okd|ocp)-.*$$ ]]; then \
		while ! $(KUBECTL) get securitycontextconstraints; do sleep 1; done; \
fi

if [[ "$KUBEVIRT_PROVIDER" =~ kind ]]; then
	echo 'Ensuring D-Bus and netplan are installed on kind nodes for netplan backend'
	for node in $(./cluster/kubectl.sh get nodes --no-headers | awk '{print $1}'); do
		echo "Installing D-Bus and netplan on ${node}"
		${CRI_BIN} exec ${node} sh -c "apt-get update -qq && apt-get install -y dbus dbus-x11 netplan.io" > /dev/null 2>&1 || true
		${CRI_BIN} exec ${node} systemctl start dbus || true
		${CRI_BIN} exec ${node} systemctl enable dbus || true
	done
	exit 0
fi

echo 'Upgrading NetworkManager and enabling and starting up openvswitch'
for node in $(./cluster/kubectl.sh get nodes --no-headers | awk '{print $1}'); do
    if [[ "$NM_VERSION" == "latest" ]]; then
        echo "Installing NetworkManager from copr networkmanager/NetworkManager-main"
        ./cluster/cli.sh ssh ${node} -- sudo dnf install -y dnf-plugins-core
        ./cluster/cli.sh ssh ${node} -- sudo dnf copr enable -y networkmanager/NetworkManager-main
    fi
    ./cluster/cli.sh ssh ${node} -- sudo dnf upgrade -y NetworkManager --allowerasing
    ./cluster/cli.sh ssh ${node} -- sudo systemctl daemon-reload
    ./cluster/cli.sh ssh ${node} -- sudo systemctl enable openvswitch
    ./cluster/cli.sh ssh ${node} -- sudo systemctl restart openvswitch
    ./cluster/cli.sh ssh ${node} -- sudo systemctl restart NetworkManager
done
