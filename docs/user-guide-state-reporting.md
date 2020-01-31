# Tutorial: Reporting State

In this example, we will use kubernetes-nmstate to report state of network
interfaces on our cluster nodes.

## Requirements

Before we start, please make sure that you have your Kubernetes/OpenShift
cluster ready with OVS. In order to do that, you can follow guides of deployment
on [local cluster](deployment-local-cluster.md) or your
[arbitrary cluster](deployment-arbitrary-cluster.md).

## Periodically report state from all nodes

The operator will periodically update the reported state of node interfaces.

Read `NodeNetworkStates` from all nodes:

```shell
kubectl get nodenetworkstates -o yaml
```

Or from a specific node:

```shell
kubectl get nodenetworkstate <node-name> -o yaml
```

The output of such command may look like this:

```yaml
apiVersion: nmstate.io/v1alpha1
kind: NodeNetworkState
metadata:
  creationTimestamp: "2019-06-24T07:48:25Z"
  generation: 1
  name: node01
  resourceVersion: "629"
  selfLink: /apis/nmstate.io/v1alpha1/nodenetworkstates/node01
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
```

If you want to learn more about the `observedState` API, see [nmstate documentation](https://nmstate.github.io/).

## Additional configuration

We can set the period of update time in seconds in config map in variable
named `node_network_state_refresh_interval`.

We can also set filter for interfaces we wish to omit in reporting
via `interfaces_filter`. This variable uses glob for pattern matching.
For example we can use values such as: `""` to keep all interfaces (disable
filtering), `"veth*"` to omit all interfaces with `veth` prefix or
`"{veth*,vnet*}"` to omit interfaces with either `veth` or `vnet` prefix.`
The default value is `"veth*"`.

These variables are controlled via a `ConfigMap`:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: nmstate-config
  namespace: nmstate
data:
  node_network_state_refresh_interval: "5"
  interfaces_filter: "veth*"
```

Please note that in order to apply changes from the `ConfigMap`, you have to
restart nmstate handler pods. That can be done by simply deleting them.
