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
The system defines 2 objects, `NodeNetworkState` and `NodeNetworkConfigurationPolicy` that
are used to report and configure cluster node networks.

For more information, please check
[kubernetes-nmstate design document](https://docs.google.com/document/d/1282BcYjYGIIxQKgMYi3nQodB4ML_gw9BSs5AXB7QUtg/),
[developer guide](docs/developer-guide.md) and
[deployment and usage section](#deployment-and-usage).

### Node Network State

Node Network State objects are created per each node in the cluster and can be
used to configure and report available interfaces and network configuration.

#### Examples

Read `NodeNetworkState` of node `node01` to list available interfaces and their
configuration:

```yaml
apiVersion: nmstate.io/v1
kind: NodeNetworkState
metadata:
  name: node01
spec:
  desiredState:
    interfaces: null
  managed: true
  nodeName: node01
status:
  currentState:
    capabilities: null
    interfaces:
    - ifIndex: 1
      name: lo
      state: down
      type: unknown
      ipv4:
        enabled: false
      ipv6:
        enabled: false
      mtu: 65536
    - ifIndex: 2
      name: eth0
      state: up
      type: ethernet
      ipv4:
        enabled: true
        dhcp: false
        address:
        - ip: 192.0.2.101
          prefix-length: 24
      ipv6:
        enabled: true
        dhcp: false
        autoConf: false
        address:
        - ip: 2001:db8::1:5001
          prefix-length: 64
      mtu: 1500
    - ifIndex: 3
      name: eth1
      state: down
      type: ethernet
      ipv4:
        enabled: false
        dhcp: false
      ipv6:
        enabled: false
        dhcp: false
        autoConf: false
      mtu: 1500
```

Apply the following `NodeNetworkState` object to configure interface `eth1` on node
`node01`:

```yaml
apiVersion: nmstate.io/v1
kind: NodeNetworkState
metadata:
  name: node01
spec:
  managed: true
  nodeName: node01
  desiredState:
    interfaces:
    - description: Production Network
      name: eth1
      state: up
      type: ethernet
      ethernet:
        auto-negotiation: true
        duplex: full
        speed: 1000
      ipv4:
        enabled: true
        dhcp: true
        address:
        - ip: 192.0.2.2
          prefix-length: 24
      ipv6:
        enabled: true
        dhcp: false
        address:
        - ip: 2001:db8::1:1
          prefix-length: 64
status:
  currentState:
    capabilities: null
    interfaces: null
```

### Node Network Configuration Policy

Node Network Configuration Policies specify the desired network configuration that
should be applied on nodes and interfaces that match given rules.

**note:** This feature has not been implemented yet and is still in design phase.

#### Examples

Apply the following `NodeNetworkConfigurationPolicy` in order to use DHCP to obtain IP
address on all `eth1` ethernet interfaces:

```yaml
apiVersion: nmstate.io/v1
kind: NodeNetworkConfigurationPolicy
metadata:
  name: eth1-dhcp-policy
spec:
  match:
    interfaces:
      name: eth1
      type: ethernet
  desiredState:
    interfaces:
    - ipv4:
        enabled: true
        dhcp: true
```

Automatically bond ethernet interfaces connected to the same VLAN 101 network:

```yaml
apiVersion: nmstate.io/v1
kind: NodeNetworkConfigurationPolicy
metadata:
  name: auto-bonding-policy
spec:
  priority: 99
  match:
    interfaces:
      type: ethernet
      lldp:
        vlan-ids: [101]
  autoConfig:
    autoBonding: true
```

## Deployment and Usage

You can choose to deploy this plugin on a
[local virtualized cluster](docs/deployment-local-cluster.md) or on your
[arbitrary cluster](docs/deployment-arbitrary-cluster.md).

After that, you can follow one of the following guides that will guide you
through node state reporting and interface configuration.

- [Report node network state](docs/user-guide-state-reporting.md)
- [Configure dummy interfaces and their MTU](docs/user-guide-state-configure-interface-mtu.md)
- [Connect an Open vSwitch bridge to a node interface](docs/user-guide-state-configure-ovs-bridge.md)

All of these can be used in both
[active and on-demand mode](docs/user-guide-active-vs-on-demand.md).

## Development and Contributing

Contributions are welcome! Find details about the project's design and
development workflow in the [developer guide](docs/developer-guide.md).
