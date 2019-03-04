# Configuration Policy Handler

**note:** This is just a design doc sketch.

Configuration policy handler can run in distributed or centralized mode. In case
of distributed (default mode), it will only handle the `NodeNetworkState`
CRDs of node it is executed on. In case of centralized mode, there has to be
only one instance of it that run at the same time.

```
+-----------------+   +-----------------+   +-----------------+
|NodeNetConfPolicy|   |NodeNetConfPolicy|   |NodeNetConfPolicy|
|                 |   |                 |   |                 |
| +------------+  |   | +------------+  |   | +------------+  |
| |match       |  |   | |match       |  |   | |match       |  |
| +------------+  |   | +------------+  |   | +------------+  |
| |autoConfig  |  |   | |autoConfig  |  |   | |autoConfig  |  |
| +------------+  |   | +------------+  |   | +------------+  |
| |desiredState|  |   | |desiredState|  |   | |desiredState|  |
+--------|--------+   +------|--|-------+   +-------|---------+
         |                   |  |                   |
         |                   |  |                   |
         |                   |  |                   |
         |                   |  |                   |
         |                   |  |                   |
+-|------v---------|-+       |  |        +-|--------v-------|-+
| |  desiredState  | |       |  |        | |  desiredState  | |
| +----------------+ <-------+  +--------> +----------------+ |
| |operationalState| |                   | |operationalState| |
| +----------------+ |                   | +----------------+ |
|                    |                   |                    |
| NodeNetworkState   |                   | NodeNetworkState   |
+---------|----------+                   +---------|----------+
          |                                        |
+---------|----------+                   +---------|----------+
|       Node         |                   |        Node        |
+--------------------+                   +--------------------+
```

## Distributed Mode

Upon invocation, it reads the list of `NodeNetworkState` CRDs, as well as the
list of `NodeNetworkConfigurationPolicy` CRDs. Instance of the policy handler
will find out on which node it is running on and it will fetch respective
`NodeNetworkState`. It will also find all `NodeNetworkConfigurationPolicy` CRDs
that apply to that node (based on their [affinity and toleration](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#affinity-and-anti-affinity)
objects).

From the interface match logic in the applicable `NodeNetworkConfigurationPolicy` CRDs,
and the list of interfaces taken from the current state of the node (in the
`NodeNetworkState` CRD), it will create aggregated desired state object, and
update it into the relevant `NodeNetworkState` CRD.
When running in "client" mode, it has nothing more to do. When running as a
daemon, it will:
- Detect updates to `NodeNetworkConfigurationPolicy` CRDs applicable for current node, and update `NodeNetworkState` CRD if needed.
- Detect an update to the current state of `NodeNetworkState` CRD for the node, and see if its desired state needs to be modified.

## Centralized mode

Very similar to the distributed mode, but in this case, the client or daemon
must handle policies and states for all nodes in the system.
