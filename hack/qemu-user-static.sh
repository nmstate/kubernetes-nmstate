#!/bin/bash

set -xe

# If qemu-static has already been registered as a runner for foreign
# binaries, for example by installing qemu-user and qemu-user-binfmt
# packages on Fedora or by having already run this script earlier,
# then we shouldn't alter the existing configuration to avoid the
# risk of possibly breaking it
if ! grep -E '^enabled$' /proc/sys/fs/binfmt_misc/qemu-aarch64 2>/dev/null; then
    ${IMAGE_BUILDER} run --rm --privileged multiarch/qemu-user-static --reset -p yes
fi
