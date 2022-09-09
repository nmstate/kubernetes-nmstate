# Host IP pooling with NNCPs.

## Summary

Provide a mechanism to assign static IP configuration for secondary interfaces
for cluster nodes, supporting cluster scaling up and down.

## Motivation

Using DHCP as the IP assignment mechanism has some drawbacks:
- Introduce latencies on node restart.
- It presents a single point of failure in case the DHCP server is failing
- It depends on network setup that can be complex.

To overcome the above-mentioned drawbacks, statically configured interfaces taking the address from a
dynamic host IP pool is a solution. Since the interfaces are static there is no
reconfiguration after some time or at node restart.

This functionality was requested on U/S via https://github.com/nmstate/kubernetes-nmstate/issues/725

Although this proposal will present a solution with an agnostic storage
mechanism, following is a spike done using whereabouts at the storage
https://github.com/nmstate/kubernetes-nmstate/pull/1081


### Similar product

- Metal3 IPAM: https://metal3.io/blog/2020/07/06/IP_address_manager.html
  - It depends on CAPI (ClusterAPI)
  - It's though for Metal3 baremetal.
- Azure private addresses: https://docs.microsoft.com/en-us/azure/virtual-network/ip-services/private-ip-addresses
  - Specific to azure
  - Some code has to be put in place to consume it.

### User Stories

- As a cluster administrator, I want to configure static IPs at non control-plane NICs on a pool of nodes without creating one per each node.
- As a cluster administrator, I want scale up the cluster with new nodes getting IPs from the policy.
- As a cluster administrator, I want each of these static IPs to be unique, and taken from a defined range, and freed when unused.

### Goals

- Allow users to use NodeNetworkConfigurationPolicy to configure static cluster-unique IP addresses without hard-coding them.

### Non-Goals

- This design does not cover IP address configuration of management NIC on day-0/1

## Proposal

### User roles

**cluster admin** is a user responsible for managing the cluster node
networking.

### Workflow Description (assign static address with IPPool)

1. The cluster admin creates `NodeIPPool` resources with IP ranges and a name.
2. The cluster admin configures node network interfaces statically, with IPs from the `NodeIPPool` using a
`NodeNetworkConfigurationPolicy` referencing them.
3. The cluster admin waits for the `NodeNetworkConfigurationPolicy` to succeed

#### Variation (exhausted range at the NodeIPPool)

When the IP pool does not have an available IP address, the `NNCP` will fail
with a explicit message, the `NodeIPPool` has to be re-created with a bigger
range.

### Workflow Description (removing a reference to NodeIPPool by NNCP re-creation)

1. The cluster admin deletes and creates the NNCP (NNCP with captures
   are not allowed to be modified).
2. The cluster admin waits for the NodeNetworkConfigurationPolicy to succeed.
   the interfaces will have re-allocated IPs.

### Workflow Description (scaling up cluster)

1. The cluster admin adds a new node affected by the `NodeNetworkConfigurationPolicy`.
2. The cluster admin waits for the `NodeNetworkConfigurationPolicy` to succeed, causing the new node to take the address from the `NodeIPPool`

### Workflow Description (scaling down cluster)

1. The cluster admin removes a node that was taking an address from a `NodeIPPool`.
2. The cluster admin waits for the `NodeNetworkConfigurationPolicy` to succeed, thus freeing the IPs from the removed node.

### Workflow Description (manual NodeIPPool clean up)

1. The cluster admin finds some stale IPs, They are inconsistencies between `NodeIPPool.status` and `NodeNetworkConfigurationEnactment.status`.
2. The cluster admin resets the `NodeIPPool` following instructions at [API Extensions/Manual Garbage Collection](#manual-garbage-collection) chapter.
3. The cluster admin waits for `NodeIPPool` status to succeed

**NOTE**: This is a manual garbage collector that is good enough for first implementation it can be converted into automatic in the future.

#### Variation (manual NNCP/NNCE allocated ips clean up)

The mechanism is the same but the pool has to be specified on the annotation

### API Extensions

This proposal adds a new CRD to configure and store IP pools (NodeIPPool) and some
changes at the NNCPs to reference the ippool.

Following is an example CR for the cluster wide CRD `NodeIPPool`:

```yaml
apiVersion: nmstate.io/v1
kind: NodeIPPool
metadata:
  name: traffic-1
spec:
  range:
    cidr: 10.10.10.0/24
    start: 10.10.10.1
    end: 10.10.10.10
    exclude: ["10.10.10.4", "10.10.10.7"]
status:
  allocated-offsets: [0, 1, 2, 4, 5, 7]
  conditions:
  - type: Available
    status: True
    reason: ReconciledPool
    message: The pool is up to date
apiVersion: nmstate.io/v1
kind: NodeIPPool
metadata:
  name: traffic-2
spec:
  range:
    cidr: 10.10.11.0/24
    start: 10.10.11.1
    end: 10.10.10.11.10
    exclude: ["10.10.11.4", "10.10.11.7"]
status:
  allocated-offsets: [0, 1, 2, 4, 5, 7]
  conditions:
  - type: Available
    status: True
    reason: ReconciledPool
    message: The pool is up to date
```

The resource does not have a namespace since it's cluster wide information
similar to NNCP. The `spec.range.cidr` is an IP range on CIDR format to
configure IP pool, `spec.range.start` is the begining of the range and
`spec.range.end` is the end, there is also an `spec.range.exclude` list of
IPs or CIDR that will not be allocated.

The `status.allocated-offsets` store the allocated IPs represented as a
list of offsets from the range.

#### Examples for allocated-offsets and their IP correlation

IPv4 /24 CIDR pool (netmask 255.255.255.0)
```yaml
cidr: 10.10.11.0/24
start: 10.10.11.4
end: 10.10.10.11.10
exclude: ["10.10.11.5", "10.10.11.7"]
```

offset -> IP:
- 0 -> 10.10.11.4
- 1 -> 10.10.11.5 (excluded)
- 2 -> 10.10.11.6
- 3 -> 10.10.11.7 (excluded)
- 4 -> 10.10.11.8
- 5 -> 10.10.11.9
- 6 -> 10.10.11.10

IPv4 /16 CIDR pool (netmask 255.255.0.0)
```yaml
cidr: 10.10.11.0/16
start: 10.10.11.4
end: 10.10.10.12.10
exclude: ["10.10.11.5", "10.10.12.7"]
```

offset -> IP:
- 0 -> 10.10.11.4
- 1 -> 10.10.11.5 (excluded)
...
- 250 -> 10.10.11.254
- 251 -> 10.10.12.1
- 252 -> 10.10.12.2
...
- 257 -> 10.10.12.7 (excluded)
- 258 -> 10.10.12.8
- 259 -> 10.10.12.9
- 260 -> 10.10.12.10

IPv4 /116 CIDR pool (4096 IPs)
```yaml
cidr: 2001:4860:4860::/116
start: 2001:4860:4860::000F
end: 2001:4860:4860::00FF
exclude: ["2001:4860:4860::0010", "2001:4860:4860::001F"]
```

offset -> IP:
- 0 -> 2001:4860:4860::000F
- 1 -> 2001:4860:4860::0010 (excluded)
...
- 17 -> 2001:4860:4860::001F (excluded)
- 18 -> 2001:4860:4860::0020
...
- 239 -> 2001:4860:4860::00FF

#### The NodeIPPool conditions

Following is a table with the "type=Available" condition states

| reason              | status  | message |
| --------------------| --------| ------ |
| ReconcilingPool     | Unknown | The pool is being updated |
| ExhaustedIPPool     | False   | The IP pool has being depleted, manual intervention needed to modify NodeIPPool.|
| UnsyncronizedIPPool | False   | Failed re constructing the IP pool, it may be not in sync with nodes. |
| OverlappingIPPool    | False   | The NodeIPPool XYZ is overlapping with this one. |
| IPCollision         | False   | The IP address XYZ at node XYZ and interface XYZ is is colliding with node XYZ |




### Referencing the NodeIPPool at NNCPs

To assign an IP address from a pool to an interface using NNCP, a special
capture syntax is introduced understood by kubernetes-nmstate
`ippool.[name].allocate` that will trigger the ip allocation mechanism.
It will inject the result on the captured state passed to nmpolicy so it can be
consumed by it, the `ippool.[name].allocate` tag will return a struct with the
following fields `ip` and `prefix-length`.

```yaml
apiVersion: nmstate.io/v1
kind: NodeNetworkConfigurationPolicy
metadata:
  name: static-ip
capture:
    address-from-traffic-1: ippool.traffic-1.allocate
    address-from-traffic-2: ippool.traffic-2.allocate
desiredState:
    interfaces:
    - name: eth1
      type: ethernet
      state: up
      ipv4:
        address:
        - ip: "{{ capture.address-from-traffic-1.ip }}"
          prefix-length: "{{ capture.address-from-traffic-1.prefix-length }}"
        dhcp: false
        enabled: true
    - name: eth2
      type: ethernet
      state: up
      ipv4:
        address:
        - ip: "{{ capture.address-from-traffic-2.ip }}"
          prefix-length: "{{ capture.address-from-traffic-2.prefix-length }}"
        dhcp: false
        enabled: true
```

The allocated IPs will be marked at the `NodeNetworkConfigurationEnactment`
status, and will be used to re-construct the nmstate IPPools on reboot.

It is also used to discover new allocations or to de-allocate them if the pool is no longer
referenced. Each node handler manages its enactment.

The `NNCE.status.ippool` information allows IP re-reservation if the
`NodeIPPool` is re-created after exhaustion, since fields on the `NodeIPPool.spec` section
are read-only for simplicity.

If the node is Ready,  the nmstate will attempt to get a new IP allocation for the desiredState.

```yaml
apiVersion: nmstate.io/v1
kind: NodeNetworkConfigurationEnactment
metadata:
  name: static-ip.node01
status:
  desiredState:
    interfaces:
    - name: eth1
      type: ethernet
      state: up
      ipv4:
        address:
        - ip: 10.10.10.8
          prefix-length: 24
        dhcp: false
        enabled: true
    - name: eth2
      type: ethernet
      state: up
      ipv4:
        address:
        - ip: 10.10.11.9
          prefix-length: 24
        dhcp: false
        enabled: true
  capturedStates:
    address-from-traffic-1:
      ip: 10.10.10.8
      prefix-length: 24
    address-from-traffic-1:
      ip: 10.10.11.9
      prefix-length: 24
  ippool:
    traffic-1:
      allocated:
      - 10.10.10.8
    traffic-2:
      allocated:
      - 10.10.11.9
```

#### Manual garbage collection

To reset the whole IP pool the user can anotate the pool with
`nmstate.io/reset-ippool`,If the `traffic1` pool need to be reset at a NNCP,
user can annotate it with `nmstate.io/reset-ippool: traffic1` and if is a
ippool `traffic1` for a specific node the NNCE can be annotated with
`nmstate.io/reset-ippool: traffic1`.

Triggering garbage collection `NodeIPPool` and `allocated-offsets` will be
re-calculated. In this process the reason will be "ReconcilingPool" and
`Available` marked as unknown, so users can wait on it.

### Implementation Details/Notes/Constraints

#### Distributed architecture

Since kubernetes-nmstate has a distributed architecture each of the handler
pods will calculate an allocatable ip from the `NodeIPPool` status and
retry the process on conflict

Handlers will check NNCEs at Reconcile and if the ip has being already assigned
it will re-use that and update `NodeIPPool` status with it, this way the
`NodeIPPool` can be reset but IPs are kept.

#### Storage optimization

To optimize on storage and lookup instead of using a list of allocated offset
it's possible to use a bitmap, the OVN project use that approach to store the
offsets on a range [1], a snapshot of the bitmap can be stored as base64 at the
`NodeIPPool` status.

[1] https://github.com/ovn-org/ovn-kubernetes/blob/master/go-controller/pkg/ovn/ipallocator/allocator.go

### Scale down scenario

For the scale down scenario, the current kubernetes-nmstate architecture
contains a controller for node that will Reconcile on node deletion, but since
the nodes may be drained before deletion the pod running the controller
will no longer be running, this means that the controller may need to run at
control-plane nodes and watch for all the nodes.


### Risks and Mitigations

If a bug prevents IPs deallocation the ippool can be exhausted and end up
with network reconfiguration or scaling up failures. To fix those situations
the field `status.allocated-offsets` at `NodeIPPool` can be set with an
empty list triggering a pool reconstruction, bringing back the
stale IPs. This risk gets higher for shorter IPPools.

If the `NodeIPPool` is exhausted it can be re-created with a bigger range,
kubernetes-nmstate will do its best to preserved the assigned IPs.

### Drawbacks

Updating the pool is distributed across nodes by the nmstate-handler pod, this
means that N nodes will try to update `NodeIPPool` and retry on conflict. Is
not expected the IPAM mechanism will run very often so this should not be a
problem.

Debugging the allocated offset can be a challange since it involve a convertion
from the offset to the IP is pointing too depending on `NodeIPPool` spec.

Fixing the "exhausted" issue involves re-creating the `NodeIPPool`, meaning
the interfaces may receive different IP addresses but we can
document that ranges has to be planned well in advance.

At non graceful node deletion the delete event may not be received at the
controller so their IPs will not be deallocated, in that case manual
intervention is needed.

## Implementation History
- Generic IPAM mechanism with in memory storage mechanism as either, standalone project, nmpolicy feature or kubernets-nmstate package.
- IPAM storage mechanism for kubernetes-nmstate use case (day 2)
- Expression "ippool.[name].allocate" feature either at kubernetes-nmstate or nmpolicy they have to call the created IPAM mechanism
- Integrate IPAM, nmpolicy and kubernetes-nmstate for simplest use case, allocate an address for the first time.
- Cover the deallocation scenario where the pool is unreferenced
- Cover the scale up scenario
- Cover the scale down scenario
- Cover restart scenarios
- Add automatic garbage collector for stale IPs
- Add a tool to understand `NodeIPPool` offsets

## Alternatives

An alternative is to implement directly an IPAM feature at Nmstate with a
plugin system for ippool retrieval and allocation storage so
it will use the `NodeIPPool` and `NodeNetworkConfigurationEnactment`, later on
it will be converted into proper NodeIPPool and NNCEs using different storage plugin.

The following would be a possible syntax:

```yaml
apiVersion: nmstate.io/v1
kind: NodeNetworkConfigurationPolicy
metadata:
  name: static-ip
desiredState:
    interfaces:
    - name: eth1
      type: ethernet
      state: up
      ipv4:
        address:
        - ipam:
            pool: traffic-1
        dhcp: false
        enabled: true
    - name: eth2
      type: ethernet
      state: up
      ipv4:
        address:
        - ipam:
            pool: traffic-2
        dhcp: false
        enabled: true
```

Pros:
- The complexity at kubernetes-nmstate would be only on the plugins.
- It would cover more scenarios.

Cons:
- This add more complexity to a project and this has to be allowed by their
  maintainers.
- Debugging would be more complex since we have multiple indirections.
