FROM --platform=linux/amd64 quay.io/centos/centos:stream8 AS build

ARG TARGETARCH

RUN dnf install -y golang-1.17.*

COPY . /workdir
WORKDIR /workdir

RUN --mount=type=cache,target=/root/.cache/go-build GOOS=linux GOARCH=${TARGETARCH} go build -o /manager ./cmd/handler

FROM quay.io/centos/centos:stream8

COPY --from=build /manager /usr/local/bin/manager

RUN dnf install -b -y dnf-plugins-core && \
    dnf install -b -y -x "*alpha*" -x "*beta*" nmstate && \
    dnf install -b -y iproute iputils && \
    dnf remove -y dnf-plugins-core && \
    dnf clean all

ENTRYPOINT ["manager"]
