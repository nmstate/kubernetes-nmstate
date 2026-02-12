#!/bin/bash

set -ex

source ./cluster/kubevirtci.sh
kubevirtci::install

$(kubevirtci::path)/cluster-up/up.sh

if [[ "$KUBEVIRT_PROVIDER" =~ ^(okd|ocp)-.*$$ ]]; then \
		while ! $(KUBECTL) get securitycontextconstraints; do sleep 1; done; \
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
    # Newer kubevirtci has dhclient installed so we should enforce not using it to
    # keep using the NM internal DHCP client as we always have
    ./cluster/cli.sh ssh ${node} -- sudo rm -f /etc/NetworkManager/conf.d/002-dhclient.conf
    ./cluster/cli.sh ssh ${node} -- sudo systemctl restart NetworkManager
done
