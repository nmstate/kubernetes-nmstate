#!/bin/bash -xe

dnf install -b -y dnf-plugins-core
dnf copr enable -y packit/nmstate-nmstate-3104
dnf install -b -y nmstate
