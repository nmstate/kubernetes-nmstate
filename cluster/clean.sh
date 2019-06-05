#!/bin/bash -e

echo 'Cleaning up ...'

./cluster/kubectl.sh delete --ignore-not-found -f build/_output/
./cluster/kubectl.sh delete --ignore-not-found -f deploy/
./cluster/kubectl.sh delete --ignore-not-found -f deploy/crds

echo 'Done'
