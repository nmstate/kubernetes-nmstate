# Development Helpers

This document serves as a reference point to development helpers available under
this project.

## Building

```shell
# If pkg/apis/ has been changed, run generator to update client code
make gen-k8s

# Build handler operator (binary and docker)
make handler

## Testing

```shell
# Run unit tests
make test/unit

# Run e2e tests
make test/e2e
```

## Containers

# Push nmstate-handler container to remote registry
make handler-push
```

It is possible to adjust the built container images with the following
environment variables.

```shell
IMAGE_REGISTRY # quay.io
IMAGE_REPO # nmstate

HANDLER_IMAGE_NAME # kubernetes-nmstate-handler
HANDLER_IMAGE_TAG # latest

```

## Manifests

The operator `operator.yaml` manifest from the `deploy` folder  is a template
to be able to replace the with correct docker image to use.

Everytime cluster-sync is called it will regenerate the operator yaml with
correct kubernets-nmstate-handler image and apply it.

```

## Local Cluster

This project uses [kubevirtci](https://github.com/kubevirt/kubevirtci) to
deploy the local cluster, the version of the repository to use is marked at
hack/install-kubevirtci.sh and it install it under `kubevirtci` dir in case
a new version is detrected the cluster command will re-install it.

Use the following commands to control it:

*note:* The default Provider is one node (master + worker) of Kubernetes 1.13.1.

```shell
# Deploy local Kubernetes cluster
export KUBEVIRT_PROVIDER=k8s-1.13.1 # k8s-1.13.1 for Kubernetes or os-3.11.0 for OpenShift
export KUBEVIRT_NUM_NODES=3 # master + two nodes
make cluster-up

# SSH to node01 and open interactive shell
kubevirtci/cluster-up/ssh.sh node01

# SSH to node01 and run command
kubevirtci/cluster-up/ssh.sh node01 -- echo 'Hello World'

# Communicate with the Kubernetes cluster using kubectl
kubevirtci/cluster-up/kubectl.sh

# Build project, build images, push them to cluster's registry and install them
make cluster-sync

# Remove all components and objects related to the project
make cluster-clean

# Destroy the cluster
make cluster-down
```

## Project Directory Structure

 ```
├── automation                  # The stdci scripts
├── cmd                         # Executable binaries
│   ├── manager                 # the operator main function command.
├── docs                        # project documentation
├── hack                        # project scripts
├── deploy                      # yaml files
│   ├── openshift               # openshift specific configuration
│   └── crds                    # CRDs configuration
├── pkg                         # libraries used by the binaries in cmd
│   ├── apis                    # CRD definitions
│   ├── helper                  # Helpers to call nmstate and change CRs
│   ├── controller              # main logic of with the operator controllers
│       ├── node                # core Node object controller
│       ├── nodenetworkstate    # NodeNetworkState controller
├── test                        # e2e tests
├── version                     # operator's version
 ```
