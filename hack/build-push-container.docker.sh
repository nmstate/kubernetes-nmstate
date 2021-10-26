#!/usr/bin/env bash

set -x -o errexit -o nounset -o pipefail

hack/init-buildx.sh

GOVERSION=$(grep "^go " go.mod |awk '{print $2}')
ARCHS=${ARCHS:-$(go env GOARCH)}
PLATFORM=""

for arch in $ARCHS; do
    PLATFORM=${PLATFORM}linux/$arch,
done

# Remove last ','
PLATFORM=${PLATFORM::-1}


PUSH=--push
if [ "$SKIP_PUSH" == "true" ]; then
    PUSH=""
fi
docker buildx build --build-arg GOVERSION=${GOVERSION} --platform ${PLATFORM} $@ -t ${IMAGE} $PUSH

