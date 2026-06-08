#!/bin/bash -xe

dnf install -b -y dnf-plugins-core

# The nmstate/nmstate-git Copr project publishes its builds under
# "centos-stream-9-<arch>" chroots. We must pass the chroot name explicitly
# because "dnf copr enable" auto-detects "epel-9-<arch>" on CentOS Stream 9,
# and the project only ships an "epel-9-x86_64" chroot (there is no
# "epel-9-aarch64"). Relying on auto-detection therefore breaks the arm64
# multi-arch image build with:
#   Error: It wasn't possible to enable this project.
#   Repository 'epel-9-aarch64' does not exist in project 'nmstate/nmstate-git'.
#
# The Copr project currently provides x86_64 and aarch64 builds only. For any
# other architecture (e.g. s390x, which is part of the multi-arch image build)
# there is no nmstate-git build available, so fall back to the distro nmstate
# package to keep the image build working across all architectures.
arch="$(uname -m)"
case "${arch}" in
    x86_64 | aarch64)
        dnf copr enable -y nmstate/nmstate-git "centos-stream-9-${arch}"
        dnf install -b -y nmstate
        ;;
    *)
        echo "nmstate/nmstate-git Copr has no ${arch} build; installing distro nmstate instead"
        dnf install -b -y -x "*alpha*" -x "*beta*" nmstate
        ;;
esac
