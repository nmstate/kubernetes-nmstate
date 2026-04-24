# 2. DaemonSet per-node Architecture

Date: 2026-04-24 (documents a decision made at project inception)

## Status

Accepted

## Context

Kubernetes operators typically run as Deployments with a single replica (or
a small leader-elected set). However, kubernetes-nmstate needs to configure
networking *on every node* in the cluster by interfacing with
NetworkManager over D-Bus. A centralised Deployment would need remote access
to each node's NetworkManager, which is neither practical nor secure.

## Decision

The **handler** component is deployed as a **DaemonSet** with per-node
[event filtering](https://sdk.operatorframework.io/docs/building-operators/golang/references/event-filtering/)
so that each handler pod only reconciles resources relevant to its own node.

A separate **operator** Deployment manages the lifecycle of the handler
DaemonSet and reconciles the cluster-scoped `NMState` CR.

## Consequences

- Each node runs its own handler pod with direct access to the host's
  NetworkManager, eliminating the need for remote access.
- Event filtering keeps the reconciliation load proportional to each node's
  own resources (`NodeNetworkState`, `NodeNetworkConfigurationEnactment`).
- The operator component remains a standard single-replica Deployment,
  responsible only for deploying and configuring the handler DaemonSet.
- Scaling is automatic: adding nodes to the cluster automatically schedules
  a new handler pod.
- The trade-off is higher per-node resource consumption compared to a
  centralised controller.
