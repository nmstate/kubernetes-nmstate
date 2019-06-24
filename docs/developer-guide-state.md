# NodeNetworkState controller

It's responsability is to fill-in and update NodeNetworkState currentStatus and
apply desiredStatus if present.

If a `NodeNetworkState` creation event is received it will fill-in currentState
from the node the pod is running on (using : `nmstatectl show`)

In case of `NodeNetworkState` update event with desiredState it will
apply directly the new config into the node (using : `nmstatectl set`)

## Configuration

This is done by modifying the `desiredState` object inside the
`NodeNetworkState` CRD. For example, assuming that the following file
(`node1-eth1-linux-bridge.yaml`) has the correct node name (`node1`), and that
eth1 interface exists (and is up) on that node:

```yaml
apiVersion: nmstate.io/v1
kind: NodeNetworkState
metadata:
  name: node1
spec:
  nodeName: node1
  desiredState:
    interfaces:
      - name: br1
        type: linux-bridge
        state: up
        bridge:
          options:
            stp:
              enabled: false
          port:
            - name: eth1
              stp-hairpin-mode: false
              stp-path-cost: 100
              stp-priority: 32
```

Calling:

```
kubectl apply -f node1-eth1-linux-bridge.yaml
```

Will create a linux bridge br1 connecting a port with eth1.

```
kubectl get nodenetworkstate node1 -o yaml
```

Should provide with the above `desiredState` and well as `currentState`
that will have (among other interfaces) the `br1` interface with eth1 as
one of the ports.
