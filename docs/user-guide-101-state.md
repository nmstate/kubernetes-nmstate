# 101: Reporting State

TODO In this example, we will use kubernetes-nmstate to report state of network
interfaces on our cluster nodes.

## List all handled nodes

The operator will periodically update the reported state of node interfaces.

Read `NodeNetworkStates` from all nodes:

```shell
kubectl get nodenetworkstates
```

```
output
```

You can also use short name `nns` to reach the same effect:

```shell
kubectl get nns
```

```
output
```

## Read state of a specific node

Or from a specific node. By using `-o yaml` you obtain the full object:

```shell
kubectl get nodenetworkstate node01 -o yaml
```

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

TODO mention it contains `observedState`, main sections `interfaces`, `routes` and
`dns-resolver`. More about that in the lesson TODO link.

It also contains `conditions`. Usually they should be Available, but in case something bad and unexpected happens on the node, Degraded condition will have the answer.

Since the report is updated periocally, there is `the timestamp object` which keeps the timestamp of the last successful update.

## Configure refresh interval

We can set the period of update time in seconds in config map in variable
named `node_network_state_refresh_interval`.

```shell
kubectl create configmap special-config --from-literal=special.how=very
```

In order for the config to take effect, it is needed to restart all handlers:

```shell
kubectl kill
kubectl wait for ds
```

Now will be the NNS updated only once every 30 seconds.

## Filter Out Interfaces

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

TODO change the filter, drop veth

Same as with the refresh interval, in order for the config to take effect, it is
needed to restart all handlers:

```shell
kubectl kill
kubectl wait for ds
```

Now will the NNS contain all interfaces, including veths:

```shell
kubectl get nns | grep 'name: veth'
```

```
output
```

## The Next Lesson

TODO: link the next unit
