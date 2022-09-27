#!/bin/bash -xe

dnf install -b -y dnf-plugins-core
dnf copr enable -y nmstate/nmstate-git
dnf install -b -y nmstate nmstate-plugin-ovsdb
