#!/bin/bash -xe

#dnf install -b -y -x "*alpha*" -x "*beta*" nmstate

dnf install -b -y dnf-plugins-core
dnf copr enable -y packit/nmstate-nmstate-2280
dnf install -b -y nmstate
