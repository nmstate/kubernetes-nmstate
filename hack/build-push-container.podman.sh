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

podman rmi ${IMAGE} || true
podman manifest rm ${IMAGE} || true
podman manifest create ${IMAGE}
IMAGES=${IMAGE}
for arch in $ARCHS; do
    podman build --arch $arch --build-arg TARGETARCH=$arch -t $IMAGE.$arch $@
    podman manifest add --tls-verify=$TLS_VERIFY ${IMAGE} docker://${IMAGE}.$arch

    if [ ! "$SKIP_PUSH" == "true" ]; then
        podman push --tls-verify=$TLS_VERIFY ${IMAGE}.$arch
    fi
done

if [ ! "$SKIP_PUSH" == "true" ]; then
    podman manifest push --tls-verify=$TLS_VERIFY ${IMAGE} docker://${IMAGE}
fi
