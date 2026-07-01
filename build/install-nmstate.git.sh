#!/bin/bash -xe

dnf install -b -y dnf-plugins-core

# Specify the centos-stream-9 chroot explicitly because dnf copr auto-detection
# picks epel-9-<arch> on CentOS Stream 9, which does not exist for aarch64 in
# the nmstate/nmstate-git Copr project.
dnf copr enable -y nmstate/nmstate-git "centos-stream-9-$(uname -m)"

dnf install -b -y nmstate
