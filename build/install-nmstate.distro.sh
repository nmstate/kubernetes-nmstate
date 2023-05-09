#!/bin/bash -xe

#dnf install -b -y -x "*alpha*" -x "*beta*" nmstate

dnf install -b -y dnf-plugins-core
# https://github.com/nmstate/nmstate/pull/2338
dnf copr enable -y packit/nmstate-nmstate-2338
dnf install -b -y nmstate
