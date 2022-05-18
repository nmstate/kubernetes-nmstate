#!/usr/bin/env bash

set -x -o errexit -o nounset -o pipefail

if [ "$SKIP_IMAGE_BUILD" == "true" ]; then
    echo "skipping image build"
    exit 0
fi

TLS_VERIFY=true
if [[ $IMAGE_REGISTRY =~ localhost ]]; then
    TLS_VERIFY=false
fi

ARCHS=${ARCHS:-$(go env GOARCH)}

buildah rmi ${IMAGE} 2>/dev/null || true
buildah manifest rm ${IMAGE} 2>/dev/null || true
buildah manifest create ${IMAGE}
IMAGES=${IMAGE}
for arch in $ARCHS; do
    buildah bud \
        --manifest ${IMAGE} \
        --arch ${arch} --build-arg TARGETARCH=${arch} $@ --tag ${IMAGE}.${arch}
done

if [ ! "$SKIP_PUSH" == "true" ]; then
    buildah manifest push --all \
        ${IMAGE} \
        docker://${IMAGE} --tls-verify=${TLS_VERIFY}
fi
