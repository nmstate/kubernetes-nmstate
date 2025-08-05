#!/bin/bash -xe

dnf install -b -y -x "*alpha*" -x "*beta*" nmstate
dnf install -y https://people.redhat.com/fge/RHEL-88896/nmstate-2.2.45-0.20250430.2514git7b1ac4b4.el9.x86_64.rpm
