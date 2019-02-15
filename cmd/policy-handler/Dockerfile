FROM golang:1.10 AS builder
WORKDIR /go/src/github.com/nmstate/kubernetes-nmstate/
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/policy-handler github.com/nmstate/kubernetes-nmstate/cmd/policy-handler

FROM centos:7
RUN yum -y update && \
    yum clean all
COPY --from=builder /bin/policy-handler /
ENTRYPOINT ["/policy-handler"]
