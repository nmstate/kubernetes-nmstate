# kubernetes-nmstate

Node-networking configuration driven by Kubernetes and executed by
[nmstate](https://nmstate.github.io/).

## The "Why"

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

## The "How"

We use [nmstate](https://nmstate.github.io/) to perform state driven network
configuration on each node, as well as to return back its current state.
This operator is driven by two types of objects, `NodeNetworkState` and
`NodeNetworkConfigurationPolicy`.

### Node Network State

`NodeNetworkState` objects are created per each node in the cluster and can be
used to report available interfaces and network configuration. These objects
are created by kubernetes-nmstate and must not be touched by a user.

Example of `NodeNetworkState` listing network configuration of node01, the full
object can be found at [Node Network State tutorial](docs/user-guide-state-reporting.md):

```yaml
apiVersion: nmstate.io/v1alpha1
kind: NodeNetworkState
metadata:
  name: node01
status:
  currentState:
    interfaces:
    - name: eth0
      type: ethernet
      state: up
      mac-address: 52:55:00:D1:55:01
      mtu: 1500
      ipv4:
        address:
        - ip: 192.168.66.101
          prefix-length: 24
        dhcp: true
        enabled: true
    ...
```

### Node Network Configuration Policy

`NodeNetworkConfigurationPolicy` objects can be used to specify desired
networking state per node or set of nodes. It uses API similar to
`NodeNetworkState`.

Example of a `NodeNetworkConfigurationPolicy` creating Linux bridge `br1` on top
of `eth1` in all the nodes in the cluster:

```yaml
apiVersion: nmstate.io/v1alpha1
kind: NodeNetworkConfigurationPolicy
metadata:
  name: br1-eth1-policy
spec:
  desiredState:
    interfaces:
    - name: br1
      description: Linux bridge with eth1 as a port
      type: linux-bridge
      state: up
      bridge:
        options:
          stp:
            enabled: false
        port:
        - name: eth1
```

## Deployment and Usage

You can choose to deploy this plugin on a
[local virtualized cluster](docs/deployment-local-cluster.md) or on your
[arbitrary cluster](docs/deployment-arbitrary-cluster.md).

After that, you can follow one of the following guides that will guide you
through node state reporting and interface configuration.

- [Report node network state](docs/user-guide-state-reporting.md)
- [Connect a Linux bridge to a node interface](docs/user-guide-policy-configure-linux-bridge.md)
- [Configure an Open vSwitch bridge to a node interface](docs/user-guide-policy-configure-ovs-bridge.md)
- [Configure a Linux bonding interface](docs/user-guide-policy-configure-linux-bond.md)

## Development and Contributing

Contributions are welcome! Find details about the project's design and
development workflow in the [developer guide](CONTRIBUTING.md).
