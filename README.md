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

#### Examples

Read `NodeNetworkState` of node `node01` to list available interfaces and their
configuration:

```yaml
apiVersion: v1
items:
- apiVersion: nmstate.io/v1
  kind: NodeNetworkState
  metadata:
    creationTimestamp: "2019-06-24T07:48:25Z"
    generation: 1
    name: node01
    resourceVersion: "629"
    selfLink: /apis/nmstate.io/v1/nodenetworkstates/node01
    uid: 7298d496-9654-11e9-ba95-525500d15501
  spec:
    managed: false
    nodeName: node01
  status:
    currentState:
      dns-resolver:
        config:
          search: []
          server: []
        running:
          search: []
          server:
          - 192.168.66.2
      interfaces:
      - bridge:
          options:
            group-forward-mask: 0
            mac-ageing-time: 300
            multicast-snooping: true
            stp:
              enabled: false
              forward-delay: 15
              hello-time: 2
              max-age: 20
              priority: 32768
          port: []
        ipv4:
          address:
          - ip: 10.244.0.1
            prefix-length: 24
          dhcp: false
          enabled: true
        ipv6:
          address:
          - ip: fe80::482b:43ff:fe71:7b87
            prefix-length: 64
          autoconf: false
          dhcp: false
          enabled: true
        mac-address: 0A:58:0A:F4:00:01
        mtu: 1450
        name: cni0
        state: up
        type: linux-bridge
      - bridge:
          options:
            group-forward-mask: 0
            mac-ageing-time: 300
            multicast-snooping: true
            stp:
              enabled: false
              forward-delay: 15
              hello-time: 2
              max-age: 20
              priority: 32768
          port: []
        ipv4:
          address:
          - ip: 172.17.0.1
            prefix-length: 16
          dhcp: false
          enabled: true
        ipv6:
          autoconf: false
          dhcp: false
          enabled: false
        mac-address: 02:42:9C:5D:73:B1
        mtu: 1500
        name: docker0
        state: up
        type: linux-bridge
      - ipv4:
          address:
          - ip: 192.168.66.101
            prefix-length: 24
          auto-dns: true
          auto-gateway: true
          auto-routes: true
          dhcp: true
          enabled: true
        ipv6:
          address:
          - ip: fe80::5055:ff:fed1:5501
            prefix-length: 64
          autoconf: false
          dhcp: false
          enabled: true
        mac-address: 52:55:00:D1:55:01
        mtu: 1500
        name: eth0
        state: up
        type: ethernet
      - ipv4:
          address: []
          auto-dns: true
          auto-gateway: true
          auto-routes: true
          dhcp: true
          enabled: true
        ipv6:
          address: []
          auto-dns: true
          auto-gateway: true
          auto-routes: true
          autoconf: true
          dhcp: true
          enabled: true
        mac-address: "52:54:00:12:34:56"
        mtu: 1500
        name: eth1
        state: down
        type: ethernet
      - ipv4:
          enabled: false
        ipv6:
          enabled: false
        mac-address: EA:0B:A9:20:CF:DC
        mtu: 1450
        name: flannel.1
        state: down
        type: unknown
      - ipv4:
          enabled: false
        ipv6:
          enabled: false
        mtu: 65536
        name: lo
        state: down
        type: unknown
      - ipv4:
          enabled: false
        ipv6:
          enabled: false
        mac-address: 0A:F0:1C:B9:08:F2
        mtu: 1450
        name: veth1dff4e1e
        state: down
        type: ethernet
      - ipv4:
          enabled: false
        ipv6:
          enabled: false
        mac-address: EE:61:20:58:66:3A
        mtu: 1450
        name: veth812db8a0
        state: down
        type: ethernet
      - ipv4:
          enabled: false
        ipv6:
          enabled: false
        mac-address: 82:40:08:CB:ED:30
        mtu: 1450
        name: vethb7811613
        state: down
        type: ethernet
      - ipv4:
          enabled: false
        ipv6:
          enabled: false
        mac-address: 2E:8E:F1:AE:B0:21
        mtu: 1450
        name: vethc5fd44f6
        state: down
        type: ethernet
      routes:
        config: []
        running:
        - destination: 10.244.0.0/24
          metric: 0
          next-hop-address: ""
          next-hop-interface: cni0
          table-id: 254
        - destination: 172.17.0.0/16
          metric: 0
          next-hop-address: ""
          next-hop-interface: docker0
          table-id: 254
        - destination: 0.0.0.0/0
          metric: 100
          next-hop-address: 192.168.66.2
          next-hop-interface: eth0
          table-id: 254
        - destination: 192.168.66.0/24
          metric: 100
          next-hop-address: ""
          next-hop-interface: eth0
          table-id: 254
        - destination: fe80::/64
          metric: 256
          next-hop-address: ""
          next-hop-interface: cni0
          table-id: 254
        - destination: fe80::/64
          metric: 256
          next-hop-address: ""
          next-hop-interface: eth0
          table-id: 254
        - destination: ff00::/8
          metric: 256
          next-hop-address: ""
          next-hop-interface: cni0
          table-id: 255
        - destination: ff00::/8
          metric: 256
          next-hop-address: ""
          next-hop-interface: eth0
          table-id: 255
kind: List
metadata:
  resourceVersion: ""
  selfLink: ""

```

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
