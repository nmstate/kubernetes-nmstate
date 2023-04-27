ARG GO_VERSION=1.18
FROM --platform=linux/amd64 registry.access.redhat.com/ubi9/go-toolset:${GO_VERSION} AS build

COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=false GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o manager ./cmd/handler

FROM quay.io/centos/centos:stream9

ARG NMSTATE_SOURCE=distro

COPY --from=build /opt/app-root/src/manager /usr/local/bin/manager
COPY --from=build /opt/app-root/src/build/install-nmstate.${NMSTATE_SOURCE}.sh install-nmstate.sh

RUN ./install-nmstate.sh && \
    dnf install -b -y iproute iputils && \
    rm ./install-nmstate.sh && \
    dnf clean all

ENTRYPOINT ["manager"]
