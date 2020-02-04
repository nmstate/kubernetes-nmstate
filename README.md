# kubernetes-nmstate

[keɪ ɛn ɛm steɪt] Declarative node network configuration driven through Kubernetes API.

# How it works

We use [nmstate](https://nmstate.io/) to perform state driven network
configuration on cluster nodes and to report back their current state.
Both the configuration and reporting is controlled via Kubernetes objects.

```yaml
apiVersion: nmstate.io/v1alpha1
kind: NodeNetworkConfigurationPolicy
metadata:
  name: br1-eth0
spec:
  desiredState:
    interfaces:
    - name: br1
      type: linux-bridge
      state: up
      ipv4:
        dhcp: true
        enabled: true
      bridge:
        port:
        - name: eth0
```

The only external dependency is NetworkManager running on nodes.

# Deployment and Usage

You can choose to deploy this plugin on a
[local virtualized cluster](docs/deployment-local-cluster.md) or on your
[arbitrary cluster](docs/deployment-arbitrary-cluster.md).

Following comprehensive 101 series is the best place to start learning about all the features of this operator:

1. [State](docs/user-guide-101-reporting-state.md) -
   observe the current state of network on cluster nodes.
2. Stay tuned for more!

These ready-to-go tutorials describe how to configure various interface types:

- [Linux bonding interface](docs/user-guide-policy-configure-linux-bond.md)
- [Linux bonding interface with vlan interface](docs/user-guide-policy-configure-linux-bond-with-vlans.md)
- [Linux bridge connected to a node interface](docs/user-guide-policy-configure-linux-bridge.md)
- [Vlan and IP on a node interface](docs/user-guide-policy-configure-vlan-and-dynamic-ip.md)
- [Open vSwitch bridge connected to a node interface](docs/user-guide-policy-configure-ovs-bridge.md)

# The "Why"

With hybrid clouds, node-networking setup is becoming even more challenging.
Different payloads have different networking requirements, and not everything
can be satisfied as overlays on top of the main interface of the node (e.g.
SR-IOV, L2, other L2).
The [Container Network Interface](https://github.com/containernetworking/cni)
(CNI) standard enables different
solutions for connecting networks on the node with pods. Some of them are
[part of the standard](https://github.com/containernetworking/plugins), and there are
others that extend support for [Open vSwitch bridges](https://github.com/kubevirt/ovs-cni),
[SR-IOV](https://github.com/hustcat/sriov-cni), and more...

However, in all of these cases, the node must have the networks setup before the
pod is scheduled. Setting up the networks in a dynamic and heterogenous cluster,
with dynamic networking requirements, is a challenge by itself - and this is
what this project is addressing.

# Development and Contributing

Contributions are welcome! Find details about the project's design and
development workflow in the [developer guide](CONTRIBUTING.md).
