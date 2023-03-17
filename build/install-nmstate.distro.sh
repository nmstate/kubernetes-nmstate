#!/bin/bash -xe

dnf install -b -y -x "*alpha*" -x "*beta*" nmstate-2.2.5
