#!/usr/bin/env bash

set -x -o errexit -o nounset -o pipefail

# Default optional environment variables so the script can be run directly
# (outside of the Makefile) without tripping `set -o nounset`.
SKIP_IMAGE_BUILD=${SKIP_IMAGE_BUILD:-false}
SKIP_PUSH=${SKIP_PUSH:-false}
# Builder forwarded to hack/qemu-user-static.sh; default and export it so a
# direct (non-make) cross-arch run can still register the QEMU binfmt handlers.
export IMAGE_BUILDER=${IMAGE_BUILDER:-docker}

if [ "$SKIP_IMAGE_BUILD" == "true" ]; then
    echo "skipping image build"
    exit 0
fi

: "${IMAGE:?IMAGE environment variable must be set}"

ARCHS=${ARCHS:-$(go env GOARCH)}

# QEMU binfmt registration is only needed for Linux cross-architecture
# builds. On macOS Docker Desktop already provides the emulation, so skip
# the setup there.
if [ "${ARCHS}" != "$(go env GOARCH)" ] && [ "$(uname -s)" == "Linux" ]; then
    hack/qemu-user-static.sh
fi

PLATFORM=""

for arch in $ARCHS; do
    PLATFORM=${PLATFORM}linux/$arch,
done

# Strip the trailing ',' -- ${PLATFORM%,} is POSIX and works on macOS bash 3.2
# (unlike ${PLATFORM::-1}, which requires bash >= 4.2).
PLATFORM=${PLATFORM%,}


PUSH=--push
if [ "$SKIP_PUSH" == "true" ]; then
    PUSH=""
fi
docker buildx build --progress plain --platform ${PLATFORM} . $@ -t ${IMAGE} $PUSH

