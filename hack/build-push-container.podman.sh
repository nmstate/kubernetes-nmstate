#!/usr/bin/env bash

set -x -o errexit -o nounset -o pipefail

# Default optional environment variables so the script can be run directly
# (outside of the Makefile) without tripping `set -o nounset`.
SKIP_IMAGE_BUILD=${SKIP_IMAGE_BUILD:-false}
SKIP_PUSH=${SKIP_PUSH:-false}
# Builder forwarded to hack/qemu-user-static.sh; default and export it so a
# direct (non-make) cross-arch run can still register the QEMU binfmt handlers.
export IMAGE_BUILDER=${IMAGE_BUILDER:-podman}

if [ "$SKIP_IMAGE_BUILD" == "true" ]; then
    echo "skipping image build"
    exit 0
fi

: "${IMAGE:?IMAGE environment variable must be set}"

TLS_VERIFY=true
if [[ ${IMAGE_REGISTRY:-} =~ localhost ]]; then
    TLS_VERIFY=false
fi

ARCHS=${ARCHS:-$(go env GOARCH)}

# QEMU binfmt registration is only needed for Linux cross-architecture
# builds. On macOS the podman machine already provides the emulation, so
# skip the setup there.
if [ "${ARCHS}" != "$(go env GOARCH)" ] && [ "$(uname -s)" == "Linux" ]; then
    hack/qemu-user-static.sh
fi

podman rmi ${IMAGE} 2>/dev/null || true
podman manifest rm ${IMAGE} 2>/dev/null || true
podman manifest create ${IMAGE}
for arch in $ARCHS; do
    podman build \
        --manifest ${IMAGE} \
        --arch ${arch} --build-arg TARGETARCH=${arch} $@ --tag ${IMAGE}.${arch} ./
done

if [ ! "$SKIP_PUSH" == "true" ]; then
    podman manifest push --all \
        ${IMAGE} \
        docker://${IMAGE} --tls-verify=${TLS_VERIFY}
fi
