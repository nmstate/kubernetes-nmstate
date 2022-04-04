ARG FROM=quay.io/centos/centos:stream8
FROM --platform=linux/amd64 ${FROM} AS build

ARG TARGETARCH

RUN mkdir /workdir

COPY go.mod /workdir
WORKDIR /workdir

RUN dnf install -y golang-$(sed -En 's/^go +(.*+)$/\1/p' go.mod).*

COPY . .

RUN GOOS=linux GOARCH=${TARGETARCH} go build -o /manager ./cmd/handler

ARG FROM=quay.io/centos/centos:stream8
FROM ${FROM}

COPY --from=build /manager /usr/local/bin/manager

RUN dnf install -b -y dnf-plugins-core && \
    dnf install -b -y -x "*alpha*" -x "*beta*" nmstate && \
    dnf install -b -y iproute iputils && \
    dnf remove -y dnf-plugins-core && \
    dnf clean all

ENTRYPOINT ["manager"]
