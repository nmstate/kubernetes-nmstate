FROM registry.access.redhat.com/ubi9/ubi AS build
ARG GO_VERSION=1.18

COPY . .

RUN ./build/install-go.sh ${GO_VERSION}
ENV PATH=/usr/local/go/bin/:$PATH

RUN --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=false go build -o manager ./cmd/handler

FROM quay.io/centos/centos:stream9

ARG NMSTATE_SOURCE=distro

COPY --from=build /manager /usr/local/bin/manager
COPY --from=build /build/install-nmstate.${NMSTATE_SOURCE}.sh install-nmstate.sh

RUN ./install-nmstate.sh && \
    dnf install -b -y iproute iputils && \
    rm ./install-nmstate.sh && \
    dnf clean all

ENTRYPOINT ["manager"]
