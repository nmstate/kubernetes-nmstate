#!/bin/bash -xe

curl -L https://github.com/operator-framework/operator-registry/releases/download/v1.19.5/linux-amd64-opm -o /tmp/opm
chmod 755 /tmp/opm
/tmp/opm $@
