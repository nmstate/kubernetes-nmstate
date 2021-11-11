---
---

# Introduction

[keɪ ɛn ɛm steɪt] Declarative node network configuration driven through Kubernetes API.

# How it works

We use [nmstate](https://nmstate.io/) to perform state driven network
configuration on cluster nodes and to report back their current state.
Both the configuration and reporting is controlled via Kubernetes objects.

```yaml
apiVersion: nmstate.io/v1
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

The only external dependency is NetworkManager running on nodes. See more
details in
[Compatibility documentation]({{ site.github.repository_url }}/blob/main/CONTRIBUTING.md).

Some [examples](examples.html) can be used as reference on how to configure various
configuration options

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
development workflow in the [developer guide]({{ site.github.repository_url }}/blob/main/CONTRIBUTING.md).
