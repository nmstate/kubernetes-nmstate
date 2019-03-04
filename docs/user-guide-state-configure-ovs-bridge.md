# Tutorial: Create an Open vSwitch Bridge and Connect it to a Node Interface

Use Node Network State to configure a new Open vSwitch bridge `br1` connected
to node interface `eth1`. This bridge can be later used to connect pods to
L2 network using [ovs-cni](https://github.com/kubevirt/ovs-cni).

## Requirements

Before we start, make sure that you have your Kubernetes/OpenShift cluster
ready with OVS. In order to do that, you can follow guides of deployment on
[local cluster](deployment-local-cluster.md) or your
[arbitrary cluster](deployment-arbitrary-cluster.md).

## Configure bridge

Install kubernetes-nmstate state handler in active mode (if not already done).

```shell
# on local cluster
./cluster/kubectl.sh create -f _out/manifests/state-controller-ds.yaml

# on arbitrary cluster
kubectl apply -f https://raw.githubusercontent.com/nmstate/kubernetes-nmstate/master/manifests/examples/state-controller-ds.yaml
```

Create `NodeNetworkState` object with desired configuration.

```yaml
# on local cluster
cat <<EOF | ./cluster/kubectl.sh -n nmstate-default create -f -
apiVersion: nmstate.io/v1
kind: NodeNetworkState
metadata:
  name: node01
spec:
  desiredState:
    interfaces:
    - description: OVS bridge for secondary pod networking
      name: br1
      type: ovs-bridge
      state: up
      bridge:
        options:
          fail-mode: ''
          mcast-snooping-enable: false
          rstp: false
          stp: true
        port:
        - name: eth1
          type: system
      ipv4:
        enabled: false
      ipv6:
        enabled: false
    - description: Bridge physical interface
      name: eth1
      type: ethernet
      state: up
      ipv4:
        enabled: false
      ipv6:
        enabled: false
  managed: true
  nodeName: node1
status:
  currentState:
    capabilities: null
    interfaces: null
EOF

# on arbitrary cluster
cat <<EOF | kubectl -n nmstate-default create -f -
apiVersion: nmstate.io/v1
kind: NodeNetworkState
metadata:
  name: node01
spec:
  desiredState:
    interfaces:
    - description: OVS bridge for secondary pod networking
      name: br1
      type: ovs-bridge
      state: up
      bridge:
        options:
          fail-mode: ''
          mcast-snooping-enable: false
          rstp: false
          stp: true
        port:
        - name: eth1
          type: system
      ipv4:
        enabled: false
      ipv6:
        enabled: false
    - description: Bridge physical interface
      name: eth1
      type: ethernet
      state: up
      ipv4:
        enabled: false
      ipv6:
        enabled: false
  managed: true
  nodeName: node1
status:
  currentState:
    capabilities: null
    interfaces: null
EOF
```
