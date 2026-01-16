#!/bin/bash -xe

dnf install -y epel-release
dnf install -b -y netplan systemd-networkd
