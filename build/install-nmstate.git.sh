#!/bin/bash -xe

dnf install -b -y dnf-plugins-core

arch=$(uname -m)
chroot="centos-stream-9-${arch}"

# Try to enable the nmstate-git Copr for the current architecture.
# Not all architectures have a build target in the Copr project (e.g. s390x),
# so fall back to the distro-provided nmstate when the chroot is unavailable.
if dnf copr enable -y nmstate/nmstate-git "${chroot}" 2>/dev/null; then
    echo "Enabled Copr nmstate-git for ${chroot}"
else
    echo "WARNING: Copr nmstate-git not available for ${chroot}, installing distro nmstate"
fi

dnf install -b -y nmstate
