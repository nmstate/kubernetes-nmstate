#!/bin/bash -e

echo 'Cleaning up ...'

./cluster/kubectl.sh delete --ignore-not-found -f build/_output/

echo 'Done'
