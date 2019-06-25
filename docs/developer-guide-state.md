# NodeNetworkState controller

It's responsability is to fill-in and update NodeNetworkState currentStatus and
apply desiredStatus if present.

If a `NodeNetworkState` creation event is received it will fill-in currentState
from the node the pod is running on (using : `nmstatectl show`)

In case of `NodeNetworkState` update event with desiredState it will
apply directly the new config into the node (using : `nmstatectl set`)

## Configuration

This is done by modifying the `desiredState` object inside the
`NodeNetworkState` CRD. For example, looking at `docs/demos` directory
in the kubernets-nmstate repository we have one manifest to create a linux
bridge br1 at node01 `docs/demo/docs/demos/create-br1-linux-bridge.yaml`,
it creates a br1 with eth1 as one of the ports

To change desiredState at that node we have to patch nodenetworkstate node01

```
kubectl patch nodenetworkstate node01 -p "$(cat docs/demo/create-br1-linux-bridge.yaml)"
```

Will create a linux bridge br1 connecting a port with eth1.

```
kubectl get nodenetworkstate node01 -o yaml
```

Should provide with the above `desiredState` and well as `currentState`
that will have (among other interfaces) the `br1` interface with eth1 as
one of the ports.
