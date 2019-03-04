# Design

The system defines 2 objects, `NodeNetworkState` and `NodeNetworkConfigurationPolicy` that
are used to report and configure cluster node interfaces. You can find more
information in [kubernetes-nmstate design document](https://docs.google.com/document/d/1282BcYjYGIIxQKgMYi3nQodB4ML_gw9BSs5AXB7QUtg/).

Each of these objects has a dedicated handler. Both of these handlers can function
in [active and on-demand modes](user-guide-active-vs-on-demand.md).

## Components

- [State Handler](developer-guide-state.md) - developer guide and design of `NodeNetworkState` handler.
- [Configuration Policy Handler](developer-guide-configuration-policy.md) - developer guide and design of `NodeNetworkConfigurationPolicy` handler.

# Development

[Development guide](developer-guide-commands.md) is a go-to reference point for
development helper commands, building, testing, container images and local
cluster.
