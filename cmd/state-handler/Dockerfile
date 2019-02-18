FROM golang:1.10 AS builder
WORKDIR /go/src/github.com/nmstate/kubernetes-nmstate/
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/state-handler github.com/nmstate/kubernetes-nmstate/cmd/state-handler

FROM centos:7
RUN yum -y install epel-release && \
    yum -y update && \
    yum -y install \
        python2-pip \
        git \
        python2-pyyaml \
        python2-six \
        python-gobject-base \
        python-jsonschema \
        python-setuptools \
        NetworkManager-libnm \
        && \
    yum clean all
RUN pip install git+https://github.com/nmstate/nmstate@d2afe631c39de5b007815789e12a2e6dab447efb
COPY --from=builder /bin/state-handler /
ENTRYPOINT ["/state-handler"]
