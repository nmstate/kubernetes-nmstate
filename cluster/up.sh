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
    ./cluster/cli.sh ssh ${node} -- sudo dnf upgrade -y NetworkManager
    ./cluster/cli.sh ssh ${node} -- sudo systemctl daemon-reload
    ./cluster/cli.sh ssh ${node} -- sudo systemctl enable openvswitch
    ./cluster/cli.sh ssh ${node} -- sudo systemctl restart openvswitch
    ./cluster/cli.sh ssh ${node} -- sudo systemctl restart NetworkManager
done
