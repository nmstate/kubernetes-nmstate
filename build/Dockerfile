ARG FROM=quay.io/centos/centos:stream9
FROM --platform=linux/amd64 ${FROM} AS build

ARG TARGETARCH

RUN mkdir /workdir

COPY go.mod /workdir
WORKDIR /workdir

RUN rpm --import /etc/pki/rpm-gpg/RPM-GPG-KEY-centosofficial && \
    dnf install -y golang-$(sed -En 's/^go +(.*+)$/\1/p' go.mod).*

COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build GOOS=linux GOARCH=${TARGETARCH} go build -o /manager ./cmd/handler

ARG FROM=quay.io/centos/centos:stream9
FROM ${FROM}

ARG NMSTATE_SOURCE=distro

COPY --from=build /manager /usr/local/bin/manager
COPY --from=build /workdir/build/install-nmstate.${NMSTATE_SOURCE}.sh install-nmstate.sh

RUN rpm --import /etc/pki/rpm-gpg/RPM-GPG-KEY-centosofficial && \
    ./install-nmstate.sh && \
    dnf install -b -y NetworkManager iproute iputils && \
    rm ./install-nmstate.sh && \
    dnf clean all

ENTRYPOINT ["manager"]
