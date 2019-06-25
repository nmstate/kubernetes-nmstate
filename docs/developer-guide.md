# Design

The system is implemented as an k8s operator using the [operator-sdk](https://github.com/operator-framework/operator-sdk)
but is deployed as a DaemonSet instead of Deployment with [filtering](https://github.com/operator-framework/operator-sdk/blob/master/doc/user/event-filtering.md) only events for the DaemonSet pod node.

There are two controllers one for [Node](https://godoc.org/k8s.io/api/core/v1#Node)
and the other for the CRD NodeNeworkState.

## Components

- [Node Controller](developer-guide-node.md) - developer guide and design of `Node` handler.
- [NodeNetworkState Controller](developer-guide-state.md) - developer guide and design of `NodeNetworkState` handler.

# Development

[Development guide](developer-guide-commands.md) is a go-to reference point for
development helper commands, building, testing, container images and local
cluster.
