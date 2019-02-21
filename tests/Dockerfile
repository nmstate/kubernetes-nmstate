FROM golang:1.10 AS builder
WORKDIR /go/src/github.com/nmstate/kubernetes-nmstate/
RUN go get github.com/onsi/ginkgo/ginkgo
COPY . .
RUN CGO_ENABLED=0 GOOS=linux ginkgo build tests
RUN mv tests/tests.test /bin/tests

FROM centos:7
COPY --from=builder /bin/tests /
ENTRYPOINT ["/tests"]
