# Node controller

It's the owner of NodeNetworkState so is the only one creating and deleting
them.

It listen to the following events:
- Node creation/deletion
- refresh timer

At Node creation/delete if creates/deletes NodeNetworkState and at timer
timing out it checks that NodeNetworkState has not been accidentally deleted
and re-create it.

After we deploy kubernetes-nmstate all nodes are reported by k8s to the
operator and new NodeNetworkState's per node will be created.
