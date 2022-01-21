ARG FROM=quay.io/centos/centos:stream8
FROM --platform=linux/amd64 ${FROM} AS build

ARG TARGETARCH

RUN mkdir /workdir

COPY go.mod /workdir

WORKDIR /workdir

RUN dnf install -b -y dnf-plugins-core && \
    dnf copr enable -y nmstate/nmstate-git && \
    dnf install -b -y make golang-$(sed -En 's/^go +(.*+)$/\1/p' go.mod).* nmstate-devel

COPY . .

RUN GOOS=linux GOARCH=${TARGETARCH} go build -o /manager ./cmd/handler

ARG FROM=quay.io/centos/centos:stream8
FROM ${FROM}

ARG NMSTATE_SOURCE=distro

COPY --from=build /manager /usr/local/bin/manager
COPY --from=build /workdir/build/install-nmstate.${NMSTATE_SOURCE}.sh install-nmstate.sh

RUN ./install-nmstate.sh && \
    dnf install -b -y iproute iputils && \
    rm ./install-nmstate.sh && \
    dnf clean all

ENTRYPOINT ["manager"]
