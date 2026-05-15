#!/bin/bash

set -xe

# Determine which foreign architectures we need QEMU for.
# On amd64 hosts building arm64 images, we need qemu-aarch64.
# On arm64 hosts building amd64 images, we need qemu-x86_64.
HOST_ARCH=$(uname -m)

case "$HOST_ARCH" in
    x86_64)  QEMU_BINFMT="qemu-aarch64" ;;
    aarch64) QEMU_BINFMT="qemu-x86_64" ;;
    *)       QEMU_BINFMT="qemu-aarch64" ;;
esac

# If qemu-static has already been registered as a runner for foreign
# binaries, for example by installing qemu-user and qemu-user-binfmt
# packages on Fedora or by having already run this script earlier,
# then we shouldn't alter the existing configuration to avoid the
# risk of possibly breaking it
if ! grep -E '^enabled$' /proc/sys/fs/binfmt_misc/${QEMU_BINFMT} 2>/dev/null; then
    if [[ "$HOST_ARCH" == "aarch64" ]]; then
        # On arm64 hosts, use tonistiigi/binfmt which provides multi-arch images,
        # unlike multiarch/qemu-user-static which is amd64-only.
        ${IMAGE_BUILDER} run --rm --privileged docker.io/tonistiigi/binfmt --install all
    else
        # On amd64 hosts, use the original multiarch/qemu-user-static.
        ${IMAGE_BUILDER} run --rm --privileged docker.io/multiarch/qemu-user-static --reset -p yes
    fi
fi
