#!/bin/bash -e

echo 'Cleaning up ...'

./cluster/kubectl.sh delete --ignore-not-found -f build/_output/
./cluster/kubectl.sh delete --ignore-not-found -f deploy/
./cluster/kubectl.sh delete --ignore-not-found -f deploy/crds/nmstate_v1_nodenetworkstate_cr.yaml
./cluster/kubectl.sh delete --ignore-not-found nodenetworkstate --all

echo 'Done'
