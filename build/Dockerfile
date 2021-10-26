ARG GOVERSION
FROM golang:$GOVERSION as manager

COPY go.* .
COPY ./api ./api
COPY ./cmd ./cmd
COPY ./controllers ./controllers
COPY ./pkg ./pkg
COPY ./vendor ./vendor

ENV GOPATH=""
ENV GOFLAGS=-mod=vendor

RUN go build -o manager ./cmd/handler

FROM centos:8
ARG NMSTATE_COPR_REPO
ARG NM_COPR_REPO

COPY --from=manager /go/manager /usr/local/bin/manager

RUN dnf install -b -y dnf-plugins-core && \
    dnf copr enable -y networkmanager/$NM_COPR_REPO && \
    dnf copr enable -y nmstate/$NMSTATE_COPR_REPO && \
    dnf copr enable -y nmstate/nispor && \
    dnf install -b -y nmstate iproute iputils && \
    dnf remove -y dnf-plugins-core && \
    dnf clean all

ENTRYPOINT ["manager"]
