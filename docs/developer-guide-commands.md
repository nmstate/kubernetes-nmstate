# Development Helpers

This document serves as a reference point to development helpers available under
this project.

## Building

```shell
# If pkg/apis/ has been changed, run generator to update client code
make generate

# Build binaries of all components, make format is called as a part of it
make build

# Refresh vendoring after dependencies change
make dep
```

## Testing

```shell
# Check code formatting, imports and properly generated modules
make check
```

## Containers

```shell
# Build containers of all components, binaries are built as a part of multi-stage containerized build
make docker

# Push built containers to remote registry
make docker-push
```

It is possible to adjust the built container images with the following
environment variables.

```shell
IMAGE_REGISTRY # quay.io/nmstate
IMAGE_TAG # latest
STATE_HANDLER_IMAGE # kubernetes-nmstate-state-handler
POLICY_HANDLER_IMAGE # kubernetes-nmstate-configuration-policy-handler
```

## Manifests

Manifests in the `manifests/examples/` folder are built from templates kept in
`manifests/templates/`.

```shell
# build manifests
make manifests
```

Manifest templates contain the following variables. It it possible to adjust
them my setting environment variables before calling `make manifests`.

```shell
NAMESPACE # nmstate-default
IMAGE_REGISTRY # quay.io/nmstate
IMAGE_TAG # latest
PULL_POLICY # Always
STATE_HANDLER_IMAGE # kubernetes-nmstate-state-handler
POLICY_HANDLER_IMAGE # kubernetes-nmstate-configuration-policy-handler
```

You can also specify input and output directories.

```shell
MANIFESTS_SOURCE # manifests/templates
MANIFESTS_DESTINATION # manifests/examples
```

## Local Cluster

This project uses [kubevirtci](https://github.com/kubevirt/kubevirtci) to
deploy the local cluster.

Use the following commands to control it:

*note:* The default Provider is one node (master + worker) of Kubernetes 1.11.0.

```shell
# Deploy local Kubernetes cluster
export KUBEVIRT_PROVIDER=k8s-1.11.0 # k8s-1.11.0 for Kubernetes or os-3.11.0 for OpenShift
export KUBEVIRT_NUM_NODES=3 # master + two nodes
make cluster-up

# SSH to node01 and open interactive shell
./cluster/cli.sh ssh node01

# SSH to node01 and run command
./cluster/cli.sh ssh node01 echo 'Hello World'

# Communicate with the Kubernetes cluster using kubectl
./cluster/kubectl.sh

# Build project, build images, push them to cluster's registry and install them
make cluster-sync

# Remove all components and objects related to the project
make cluster-clean

# Destroy the cluster
make cluster-down
```

## Project Directory Structure

 ```
├── cluster               # local cluster scripts
├── cmd                   # location of binaries main functions and Dockerfiles
│   ├── policy-handler
│   └── state-handler
├── docs                  # project documentation
├── hack                  # project scripts
├── manifests             # yaml files
│   ├── examples          # generated yaml files
│   └── templates         # templates for all yaml files, CRDs, daemon sets and others
├── pkg                   # libraries used by the binaries in cmd
│   ├── apis              # CRD definitions
│   ├── client            # generated code for Kubernetes API client
│   ├── nmstatectl        # conversion between CRD and nmstatectl input/output
│   ├── policy-controller # main logic of policy handler running as a daemon
│   ├── state-controller  # main logic of state handler running as a daemon
│   └── utils             # general utility functions
└── tools                 # tools for yaml generation
 ```
