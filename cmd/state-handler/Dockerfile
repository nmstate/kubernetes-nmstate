FROM golang:1.10 AS builder
WORKDIR /go/src/github.com/nmstate/kubernetes-nmstate/
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/state-handler github.com/nmstate/kubernetes-nmstate/cmd/state-handler

FROM centos:7
RUN yum -y install epel-release && \
    yum -y update && \
    yum -y install \
        https://kojipkgs.fedoraproject.org//packages/nmstate/0.0.5/1.el7/noarch/python2-libnmstate-0.0.5-1.el7.noarch.rpm \
        https://kojipkgs.fedoraproject.org//packages/nmstate/0.0.5/1.el7/noarch/nmstate-0.0.5-1.el7.noarch.rpm && \
    yum clean all
COPY --from=builder /bin/state-handler /
ENTRYPOINT ["/state-handler"]
