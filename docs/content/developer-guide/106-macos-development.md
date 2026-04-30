---
title: "macOS Development"
weight: 60
type: docs
---

This page covers developing kubernetes-nmstate on macOS using a [Lima](https://lima-vm.io/) virtual machine and an external Kubernetes cluster.

## Overview

On macOS (especially Apple Silicon), `make cluster-up` is **not supported** because kubevirtci is amd64-only and Go's runtime crashes under arm64 emulation (both Rosetta and QEMU user-static). Instead, the macOS workflow uses:

1. An **external Kubernetes cluster** running on x86_64 Linux (remote machine, cloud VM, etc.)
2. A **Lima VM** on your Mac for building images and running `make cluster-sync`

Lima provides a lightweight Linux VM with automatic file sharing — you edit code on macOS and build inside the VM transparently.

## Prerequisites

Install Lima via Homebrew:

```bash
brew install lima
```

You also need an x86_64 Linux machine with a running Kubernetes cluster. See the [Local Development Cluster]({{< relref "/developer-guide/103-local-cluster" >}}) page for how to set one up with `make cluster-up` on Linux.

## Quick Start

Set up your environment:

```bash
export KUBEVIRT_PROVIDER=external
export KUBECONFIG=/path/to/your/cluster/kubeconfig
export DEV_IMAGE_REGISTRY=<your-registry>   # e.g., quay.io/yourusername
```

Then build and deploy:

```bash
make cluster-sync
```

On macOS, `cluster-sync` detects Darwin, ensures the Lima VM is running (creating it automatically on first use), and re-executes itself inside the VM. Inside the VM, it builds container images, pushes them to your registry, and deploys to the external cluster. This is completely transparent.

## How It Works

The cluster scripts that support macOS (`cluster/sync.sh`, `cluster/clean.sh`, etc.) source `cluster/lima.sh` and call `lima::ensure_linux` at the top. This function:

1. Checks if running on macOS (on Linux, it returns immediately — zero overhead)
2. Verifies Lima is installed
3. Creates the VM from `lima/kubernetes-nmstate.yaml` if it doesn't exist
4. Starts the VM if it's stopped
5. Re-executes the same script inside the VM via `limactl shell`, forwarding all relevant environment variables (including `KUBEVIRT_PROVIDER`, `KUBECONFIG`, `DEV_IMAGE_REGISTRY`)

Scripts that require kubevirtci (`cluster/up.sh`, `cluster/down.sh`) detect macOS and exit with a helpful error message instead.

## Workflow

- **Edit code** using your macOS editor or IDE — changes are immediately visible inside the VM via Lima's shared mount
- **Run `make cluster-sync`** from your macOS terminal to build and deploy
- **Run `make cluster-clean`** to clean up deployed resources
- **Run unit tests** inside the VM: `limactl shell kubernetes-nmstate` then `make test/unit`
- **Manage the external cluster** (up/down) directly on the Linux machine

## Configuration

### VM Name

The default Lima VM name is `kubernetes-nmstate`. Override it with:

```bash
export LIMA_VM_NAME=my-custom-vm
make cluster-sync
```

### Environment Variables

All standard environment variables are forwarded into the VM automatically:

- `KUBEVIRT_PROVIDER`, `KUBECONFIG`
- `KUBEVIRT_NUM_NODES`, `KUBEVIRT_NUM_SECONDARY_NICS`
- `KUBEVIRT_DEPLOY_PROMETHEUS`, `KUBEVIRT_DEPLOY_GRAFANA`
- `NM_VERSION`, `NMSTATE_VERSION`
- `DEV_IMAGE_REGISTRY`, `IMAGE_REGISTRY`, `IMAGE_REPO`

**Important**: `KUBECONFIG` must point to a file under your home directory (`~`) since that's what Lima mounts into the VM.

## Stopping and Restarting

Stop the VM (preserves state):

```bash
limactl stop kubernetes-nmstate
```

Restart it later:

```bash
limactl start kubernetes-nmstate
```

Or just run a make target — the scripts will start the VM automatically if it's stopped.

## Cleanup

Delete the VM entirely:

```bash
limactl delete kubernetes-nmstate
```

## Troubleshooting

### Docker socket permissions

If you see `permission denied` errors when running Docker commands inside the VM, your user may not have picked up the `docker` group yet. Run:

```bash
limactl shell kubernetes-nmstate
newgrp docker
```

### Registry authentication

If `make cluster-sync` fails with `unauthorized: access to the requested resource is not authorized`, the Lima VM doesn't have your container registry credentials. macOS Docker Desktop stores credentials in the macOS Keychain via the `osxkeychain` helper, which isn't available inside the VM. Log in to your registry inside the VM:

```bash
limactl shell kubernetes-nmstate
docker login quay.io
```

You only need to do this once — the credentials are stored in `~/.docker/config.json` which persists across VM restarts.

### Disk space

If you run out of space in the VM, delete unused Docker images:

```bash
limactl shell kubernetes-nmstate
docker system prune -a
```

### Manual shell access

For debugging or running commands not covered by the transparent integration:

```bash
limactl shell kubernetes-nmstate
cd /Users/<your-user>/Developer/kubernetes-nmstate
```

## Why Not cluster-up on macOS?

kubevirtci provisions Kubernetes nodes as Docker containers running x86_64 Linux. On Apple Silicon Macs, these containers need emulation:

- **Rosetta / qemu-user-static** (user-mode emulation): Go's garbage collector crashes because its `lfstack` pointer packing assumes a 48-bit x86_64 address space, but arm64 emulation maps memory at addresses that overflow this packing.
- **QEMU full system emulation**: Works correctly but is too slow — kubevirtci nodes take 30+ minutes to boot and are unusable for development.

Until kubevirtci provides native arm64 images, an external x86_64 cluster is the recommended approach.
