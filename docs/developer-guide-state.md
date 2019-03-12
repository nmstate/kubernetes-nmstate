# State Handler

The state handler is responsible of updating `NodeNetworkState` of the node it
is running on as well as of applying desired state.

When it starts on the node, it reads the list of `NodeNetworkState` CRDs, if
no CRD exist for the node it is executed on, it will create one, and fill it
with the output of `nmstatectl show` as the current status in the CRD.

If a `NodeNetworkState` CRD exists for the node, it will try to enforce the
desired state from the CRD (using: `nmstatectl set`), and then report back
current state.

When running in "client" mode, it has nothing more to do. When running as a
daemon, it will:

- Detect an update to the `NodeNetworkState` CRD which apply to the node it is running on, then, it will try to reenforce the desired state, and report back the current one. In case it detects deletion of the `NodeNetworkState` CRD, it will re-create it with current state only.
- In case that the enforcement partially or completely failed, the daemon will retry to enforce it (with exponential back-off) until it succeeded, or the desired state is modified again 
- Even if enforcement was successful, the daemon will periodically poll the current state of the node, and will report it if any modification happened. If such modification is causing the desired state to be different than the current one, it will try to reenforce it (as described above).

> Notes:
> - The `NodeNetworkState` CRD has an "un-managed" indicator, allowing an administrator to stop all enforcement and reporting for a node.
> - The desire state could be created based on `NodeNetworkConfigurationPolicy` CRDs (see below), or just manually set by an external system.

## Configuration

This is done by modifying the `desiredState` object inside the
`NodeNetworkState` CRD. For example, assuming that the following file
(`node1-state.yaml`) has the correct node name (`node1`), and that
dummy0 interface exists (and is up) on that node:

```yaml
apiVersion: nmstate.io/v1
kind: NodeNetworkState
metadata:
  name: node1
spec:
  desiredState:
    interfaces:
    - name: dummy0
      state: up
      type: dummy
      mtu: 1450
  managed: true
  nodeName: node1
status:
  currentState:
    capabilities: null
    interfaces: null
```

Calling:

```
kubectl apply -f node1-state.yaml
```

Will set its MTU to 1450. A consequent call to:

```
kubectl get nodenetworkstate node1 -o yaml
```

Should provide with the above `desiredState` and well as `currentState`
that will have (among other interfaces) the `dummy0` interface with the new
MTU.
