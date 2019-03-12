# Active vs On-Demand Mode

Cluster Network Addons Operator project we provide 2 handlers (one per each
CRD). These handlers can be invoked in on on-demand or active mode.

## On-Demand

When launched in on-demand (client/one-shot) mode, the handler would run on
selected nodes and handle `NodeNetworkState`/`NodeNetworkConfigurationPolicy`
objects available at the moment. Once the handling is done, it would stop.

This mode could be used by an external user, e.g.
[Machine Config Operator](https://github.com/openshift/machine-config-operator).

## Active

When launched in active mode, the handler would be distributed across cluster as
a daemon set and listen for `NodeNetworkState`/`NodeNetworkConfigurationPolicy`
changes. Node changes will be reflected in `NodeNetworkState` and vice-versa.
