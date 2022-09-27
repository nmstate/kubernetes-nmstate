#!/bin/bash -xe

dnf install -b -y -x "*alpha*" -x "*beta*" nmstate nmstate-plugin-ovsdb
