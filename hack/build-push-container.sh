#!/usr/bin/env bash

set -x -o errexit -o nounset -o pipefail

if [ "$SKIP_IMAGE_BUILD" == "true" ]; then
    echo "skipping image build"
    exit 0
fi

podman version

ARCHS=${ARCHS:-$(go env GOARCH)}
PLATFORM=""

for arch in $ARCHS; do
    PLATFORM=${PLATFORM}linux/$arch,
done

# Remove last ','
PLATFORM=${PLATFORM::-1}


podman build --manifest ${IMAGE} --build-arg GO_VERSION=${GO_VERSION} --progress plain --platform ${PLATFORM} . $@
if [ "$SKIP_PUSH" != "true" ]; then
    podman manifest push --tls-verify=false ${IMAGE} docker://${IMAGE}
fi

