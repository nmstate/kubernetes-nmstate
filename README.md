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
The system defines 1 objects, `NodeNetworkState`  that
is used to report and configure cluster node networks.

For more information, please check
[kubernetes-nmstate design document](https://docs.google.com/document/d/1282BcYjYGIIxQKgMYi3nQodB4ML_gw9BSs5AXB7QUtg/),
[developer guide](docs/developer-guide.md) and
[deployment and usage section](#deployment-and-usage).

### Node Network State

Node Network State objects are created per each node in the cluster and can be
used to configure and report available interfaces and network configuration.

Example of NodeNetworkState listing network configuration of node01 can be
found in manifests/docs/demos/state.yaml.

## Deployment and Usage

You can choose to deploy this plugin on a
[local virtualized cluster](docs/deployment-local-cluster.md) or on your
[arbitrary cluster](docs/deployment-arbitrary-cluster.md).

After that, you can follow one of the following guides that will guide you
through node state reporting and interface configuration.

- [Report node network state](docs/user-guide-state-reporting.md)
- [Connect an Linux bridge to a node interface](docs/user-guide-state-configure-linux-bridge.md)

## Development and Contributing

Contributions are welcome! Find details about the project's design and
development workflow in the [developer guide](docs/developer-guide.md).
