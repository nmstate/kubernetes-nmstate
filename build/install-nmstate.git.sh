#!/bin/bash -xe

microdnf install -y dnf dnf-plugins-core
dnf copr enable -y nmstate/nmstate-git
dnf install -y nmstate
