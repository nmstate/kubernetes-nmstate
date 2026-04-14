---
title: "Getting Started"
weight: 10
type: docs
---

This page covers the prerequisites and basic building instructions for kubernetes-nmstate.

## Prerequisites

- Go 1.21 or later
- Container runtime (podman or docker)
- kubectl
- For local cluster development: [kubevirtci](https://github.com/kubevirt/kubevirtci)

## Building

### Build Commands

```bash
# Build handler container image
make handler

# Build operator container image
make operator

# Build both images
make all

# Push handler image to registry
make push-handler

# Push operator image to registry
make push-operator

# Push both images
make push
```

### Container Image Configuration

Configure container images via environment variables:

- `IMAGE_REGISTRY`: Container registry (default: quay.io)
- `IMAGE_REPO`: Repository name (default: nmstate)
- `HANDLER_IMAGE_NAME`: Handler image name (default: kubernetes-nmstate-handler)
- `HANDLER_IMAGE_TAG`: Handler image tag (default: latest)
- `OPERATOR_IMAGE_NAME`: Operator image name (default: kubernetes-nmstate-operator)
- `OPERATOR_IMAGE_TAG`: Operator image tag (default: latest)
- `IMAGE_BUILDER`: podman or docker (auto-detected)

### Building on Apple Silicon Mac

Building on Apple Silicon Macs is only supported with podman. To build amd64 with podman, set up a `podman machine` configured to run amd64 containers:

```shell
# Initialize the machine with your preferred specs
podman machine init --cpus=8 --disk-size=20 --memory 8192
podman machine start

# Once the machine is ready and started up ssh into it
podman machine ssh
sudo -i

# Install qemu-user-static (if not installed already)
rpm-ostree install qemu-user-static
systemctl reboot
```

## External Cluster Development

To develop against an external OCP cluster using a custom container registry:

Set the environment variables `KUBEVIRT_PROVIDER=external` and `KUBECONFIG` pointing to the k8s cluster config.

Use `DEV_IMAGE_REGISTRY` and `IMAGE_REPO` to specify where dev containers are pushed.

Example using quay.io/foo/ as the dev registry:

```bash
docker login -u foo quay.io
make DEV_IMAGE_REGISTRY=quay.io IMAGE_REPO=foo cluster-sync
```

## Next Steps

Once you have the build environment set up, continue to [Testing]({{< relref "/developer-guide/102-testing" >}}) to learn about running tests.
