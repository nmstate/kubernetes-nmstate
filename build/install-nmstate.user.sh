#!/bin/bash -xe

curl "https://people.redhat.com/fge/bz_2158151/nmstate_hotfix.repo" -o /etc/yum.repos.d/nmstate_hotfix.repo

dnf install -b -y -x "*alpha*" -x "*beta*" nmstate
