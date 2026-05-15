# 1. Use Architecture Decision Records

Date: 2026-04-24

## Status

Accepted

## Context

As kubernetes-nmstate evolves, architectural decisions are made in issues,
PRs, and design discussions that are difficult to discover later. New
contributors and maintainers need to understand not just *what* the code does
but *why* certain design choices were made.

## Decision

We will use Architecture Decision Records (ADRs) as described by
[Michael Nygard](https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions).

ADRs are stored in `docs/adr/` and follow a sequential numbering scheme
(`NNNN-short-title.md`). Each ADR records a single decision with its context,
rationale, and expected consequences.

## Consequences

- Architectural decisions are documented alongside the code they affect.
- Future contributors can understand the reasoning behind past choices.
- ADRs are lightweight, version-controlled, and require no special tooling.
- New decisions should add a new ADR rather than editing existing ones;
  superseded ADRs are updated with a "Superseded by" status.
