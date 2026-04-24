# 3. Use nmstate as the Network Configuration Abstraction

Date: 2026-04-24 (documents a decision made at project inception)

## Status

Accepted

## Context

Configuring node networking declaratively requires an abstraction layer
that can translate a desired-state specification into concrete
NetworkManager operations. Options considered:

1. **Direct NetworkManager D-Bus API** – full control but extremely
   low-level, requiring reimplementation of state diffing and rollback.
2. **Shell scripts / nmcli wrappers** – simple but fragile and hard to
   make idempotent.
3. **nmstate** – a purpose-built library and CLI (`nmstatectl`) that
   provides declarative, state-driven network management on top of
   NetworkManager with built-in verification and rollback.

## Decision

Use [nmstate](https://nmstate.io/) (via `nmstatectl`) as the sole
network configuration backend. The handler invokes `nmstatectl set` with
the desired state and `nmstatectl show` to report current state.

## Consequences

- Declarative networking with automatic state diffing and rollback comes
  for free from nmstate.
- The project inherits nmstate's NetworkManager version requirements
  (see the compatibility matrix in `CONTRIBUTING.md`).
- nmstate must be available in the handler container image — the project
  builds container images with nmstate bundled.
- Changes in nmstate's YAML/JSON schema may require corresponding
  updates to kubernetes-nmstate CRDs and validation webhooks.
