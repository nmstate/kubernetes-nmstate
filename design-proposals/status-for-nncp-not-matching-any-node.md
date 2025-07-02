# Status for NNCP not matching any node(s)

## Summary

This proposal is trying to figure out a way on how to handle the status response kubernetes-nmstate is reporting on
`oc get nncp -oyaml` when the NNCP is not matching any node(s).

## Motivation

Users unfamiliar with the status reports of `oc get nncp -oyaml` might confuse the status report of `status.conditions[].type`
(Available) and incorrectly assume that a NNCP has been properly applied to node(s) even though it is possibly not
matching any node(s) `status.conditions[].message` (Policy does not match any node) and `status.conditions[].reason`
(NoMatchingNode). Tickets like [these](https://issues.redhat.com/browse/OPNET-81) have already come up in the past and
should therefore be addressed to provide a clear answer.

### User Stories

- As a cluster administrator, I want to be able to easily judge if an NNCP has successfully been rolled out by the status.

### Goals

- Clarify which `status.condition[]` list an applied NNCP not matching any node(s) should have.

### Non-Goals

- Changes to `status.condition[]` on anything else than applied NNCP.

## Proposal

### Solution 1: Add new status [completed/ignored]

Adding a new status will indicate to the user that the NNCP is neither `progressing`, `degraded` nor `available`.
That status could be called `completed` or `ignored`. `Completed` could indicate that a NNCP has applied but is not "running"
therefore only `completed` and not `available`. `Ignored` could be used to indicate that the NNCP is valid therefore not
`degraded` but is being ignored by all nodes
for any reason (e.g. wrong node selectors).

#### Pros

- It is immediately visible to an admin that the NNCP has been applied successfully but has not affected any node.

#### Cons

- This proposal will add a new status. Scripts from cluster admins checking and relying on the status being (Available) will most likely
  not work anymore and therefore need some changes.

### Solution 2: Set status degraded

The status could be set to `degraded` to indicate the NNCP is not behaving as expected.

#### Pros

- It is immediately visible to an admin that the applied NNCP might not behave as expected.

#### Cons

- Setting the status to `degraded` could confuse cluster admins that rely on the status indicating an actual problem.
- It could also cause scripts of cluster admins to behave incorrectly. As they already could rely on the status being `available`
  even though the NNCP does not match any node(s).

### Solution 3: Clarify this behavior in docs

Clarifying the status being `available` even when no node(s) are selected in the docs could prevent tickets like the one
mentioned above.

#### Pros

- It is documented behavior and the impact on API is minimal and there will be no "breakage" of scripts any cluster admin
  might have.

#### Cons

- Cluster admins could overlook the entry in the docs and still ask about the issue.
- It's still not very clear to cluster admins looking at the status if the NNCP has been adopted by any node(s).

### API Changes

This proposal could change the API in a sense that it would return a potentially unknown status `completed/ignored`. Or
return a `degraded` status where a cluster admin used to the current behavior might become confused.

Current state:

```yaml
apiVersion: v1
items:
  - apiVersion: nmstate.io/v1
    kind: NodeNetworkConfigurationPolicy
    spec:
      desiredState:
        ...
      nodeSelector:
        kubernetes.io/hostname: hsot02.domain.example # <- typo in node selection
    status:
      conditions:
        - lastHearbeatTime: "2022-04-25T16:02:56Z"
          lastTransitionTime: "2022-04-25T16:02:56Z"
          message: Policy does not match any node
          reason: NoMatchingNode
          status: "True"
          type: Available
        - lastHearbeatTime: "2022-04-25T16:02:56Z"
          lastTransitionTime: "2022-04-25T16:02:56Z"
          message: Policy does not match any node
          reason: NoMatchingNode
          status: "False"
          type: Degraded
        - lastHearbeatTime: "2022-04-25T16:02:56Z"
          lastTransitionTime: "2022-04-25T16:02:56Z"
          reason: ConfigurationProgressing
          status: "False"
          type: Progressing
kind: List
metadata:
  resourceVersion: ""
  selfLink: ""
```

Solution 1:

```yaml
apiVersion: v1
items:
  - apiVersion: nmstate.io/v1
    kind: NodeNetworkConfigurationPolicy
    spec:
      desiredState:
        ...
      nodeSelector:
        kubernetes.io/hostname: hsot02.domain.example # <- typo in node selection
    status:
      conditions:
        - lastHearbeatTime: "2022-04-25T16:02:56Z"
          lastTransitionTime: "2022-04-25T16:02:56Z"
          message: Policy does not match any node
          reason: NoMatchingNode
          status: "True"
          type: Ignored
        - lastHearbeatTime: "2022-04-25T16:02:56Z"
          lastTransitionTime: "2022-04-25T16:02:56Z"
          message: Policy does not match any node
          reason: NoMatchingNode
          status: "False"
          type: Degraded
        - lastHearbeatTime: "2022-04-25T16:02:56Z"
          lastTransitionTime: "2022-04-25T16:02:56Z"
          reason: ConfigurationProgressing
          status: "False"
          type: Progressing
kind: List
metadata:
  resourceVersion: ""
  selfLink: ""
```

Solution 2:

```yaml
apiVersion: v1
items:
  - apiVersion: nmstate.io/v1
    kind: NodeNetworkConfigurationPolicy
    spec:
      desiredState:
        ...
      nodeSelector:
        kubernetes.io/hostname: hsot02.domain.example # <- typo in node selection
    status:
      conditions:
        - lastHearbeatTime: "2022-04-25T16:02:56Z"
          lastTransitionTime: "2022-04-25T16:02:56Z"
          message: Policy does not match any node
          reason: NoMatchingNode
          status: "True"
          type: Degraded
        - lastHearbeatTime: "2022-04-25T16:02:56Z"
          lastTransitionTime: "2022-04-25T16:02:56Z"
          message: Policy does not match any node
          reason: NoMatchingNode
          status: "False"
          type: Available
        - lastHearbeatTime: "2022-04-25T16:02:56Z"
          lastTransitionTime: "2022-04-25T16:02:56Z"
          reason: ConfigurationProgressing
          status: "False"
          type: Progressing
kind: List
metadata:
  resourceVersion: ""
  selfLink: ""
```

Solution 3 is exactly like the current status as it wouldnt change the response.
