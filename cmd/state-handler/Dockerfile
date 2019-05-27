FROM golang:1.10 AS builder
WORKDIR /go/src/github.com/nmstate/kubernetes-nmstate/
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/state-handler github.com/nmstate/kubernetes-nmstate/cmd/state-handler

FROM centos:7
RUN yum-config-manager --add-repo https://copr.fedorainfracloud.org/coprs/nmstate/nmstate-el7/repo/epel-7/nmstate-nmstate-el7-epel-7.repo && \
    yum -y install epel-release && \
    yum -y update && \
    yum -y install nmstate && \
    yum clean all
COPY --from=builder /bin/state-handler /
ENTRYPOINT ["/state-handler"]
