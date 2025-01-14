# Globally customize DNS cluster health probe.

## Summary

Support customize kubernetes-nmstate DNS resolving probes at NMState CR.

## Motivation

At some cluster their network configuration is not compatible with DNS health
probe the kubernetes-nmstate cluster health probes, so the NNCPs will fail.

### User Stories

- As a cluster administrator, I want to customize globally the kubernetes-nmstate DNS health probes so they work for my cluster.

### Goals

- Allow users to customize cluster health probes DNS globally using the NMState CR

### Non-Goals

- Allow user to disable DNS probe
- Allow user to customize cluster health probes per NNCP

## Proposal

### User roles

**cluster admin** is a user responsible for managing the cluster node
networking and install operators

### Workflow Description (customize DNS probe name to resolve)

1. The cluster admin configures DNS probe name to resolve at NMState CR.
2. The cluster admin creates an NNCP and kubernetes-nmstate will use the custom DNS probe name to check cluster health

### Workflow Description (go back to default DNS probe)

1. The cluster admin remove the DNS probe customization 
2. The cluster admin creates an NNCP and the default name resolution is use for the DNS probe

### Alternatives

#### Add a DNS probe fallback using global name

A possible alternative is the DNS probe in case of current implementation 
failing doing a fallback to use a global name like "redhat.com" or the like and
the golang call `LookupNetIP` this can be done before apply the NNCP to select
the proper mechanism to check after apply NNCP, it can be done at installation 
time too.

### API Extensions

This proposal add a new `probes` field under the `NMState` CR to configure the 
DNS probes name resolution, this way if the other probes need to be customize
in the future more fields can be added there

Following is the `NMState` CR example with custom DNS probe name resolution:

```yaml
apiVersion: nmstate.io/v1
kind: NMState
metadata:
  name: nmstate
spec:
  probes:
    DNS: 
      name: "redhat.com"
```

After doing this the probe will use golang `LookupNetIP` function instead of 
current `LookupNS` to resolve the specified `spec.probes.DNS.name` "redhat.com".
