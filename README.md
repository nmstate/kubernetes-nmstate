# kubernetes-nmstate

[keɪ ɛn ɛm steɪt] Declarative node network configuration driven through Kubernetes API.

## How it works

We use [nmstate](https://nmstate.io/) to perform state driven network
configuration on cluster nodes, as well as to return back their current state.
Both the configuration and reporting is controlled via Kubernetes objects. TODO.

```yaml
apiVersion: nmstate.io/v1alpha1
kind: NodeNetworkConfigurationPolicy
metadata:
  name: default-interface
spec:
  desiredState:
    interfaces:
    - name: eth0
      type: ethernet
      state: up
      ipv4:
        enabled: true
        dhcp: true
```

The only external dependency is NetworkManager running on your hosts. TODO.

## Deployment and Usage

You can choose to deploy this plugin on a
[local virtualized cluster](docs/deployment-local-cluster.md) or on your
[arbitrary cluster](docs/deployment-arbitrary-cluster.md).

Following 101 series will guide you through TODO HELP ME:

1. [State](docs/user-guide-101-state.md) -
   observe the current state of network on cluster nodes.
2. [First Policy and Enactments](docs/user-guide-102-first-policy-and-enactments.md) -
   configure networks, observe the progress and troubleshoot issues.
3. [Desired state API](docs/user-guide-103-desired-state-api.md) -
   learn how to describe interfaces, routes and DNS.
4. [Selecting nodes](docs/user-guide-104-selecting-nodes.md) -
   apply configuration only on a subset of nodes.
5. [Safe guards](docs/user-guide-105-safe-guards.md) -
   learn how is you cluster protected against invalid configuration.

These ready-to-go tutorials describe how to configure various interface types:

- [Ethernet NIC](docs/user-guide-tutorial-ethernet-nic.md)
- [Linux bridge as the default interface](docs/user-guide-tutorial-linux-bridge-as-the-default-interface.md)
- [VLAN interface](docs/user-guide-tutorial-vlan-interface.md)
- [Linux bonding](docs/user-guide-tutorial-linux-bonding.md)
- [VLAN interface over Linux bonding](docs/user-guide-tutorial-vlan-interface-over-linux-bonding.md)
- [Open vSwitch bridge as the default interface](docs/user-guide-tutorial-openvswitch-bridge-as-the-default-interface.md)

## Development and Contributing

Contributions are welcome! Find details about the project's design and
development workflow in the [developer guide](docs/developer-guide.md).
