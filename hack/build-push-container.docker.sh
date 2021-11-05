#!/usr/bin/env bash

set -x -o errexit -o nounset -o pipefail

hack/init-buildx.sh

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
docker buildx build --platform ${PLATFORM} $@ -t ${IMAGE} $PUSH

