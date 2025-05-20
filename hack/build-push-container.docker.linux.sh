#!/usr/bin/env bash

set -x -o errexit -o nounset -o pipefail

if [ "$SKIP_IMAGE_BUILD" == "true" ]; then
    echo "skipping image build"
    exit 0
fi

hack/init-buildx.sh

ARCHS=${ARCHS:-$(go env GOARCH)}

if [ "${ARCHS}" != "$(go env GOARCH)" ]; then
    hack/qemu-user-static.sh
fi

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
docker buildx build --progress plain --platform ${PLATFORM} . $@ -t ${IMAGE} $PUSH

