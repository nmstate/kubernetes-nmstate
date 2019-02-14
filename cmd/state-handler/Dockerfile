FROM golang:1.10 AS builder
WORKDIR /go/src/github.com/nmstate/kubernetes-nmstate/
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/state-handler github.com/nmstate/kubernetes-nmstate/cmd/state-handler

FROM centos:7
RUN yum -y install iproute NetworkManager-libnm git cairo* epel-release && \
    yum -y install python2-pip gcc python-devel gobject-introspection-devel && \
    yum clean all
RUN pip install -U \
      pip \
      setuptools \
      pbr \
      tox \
      pluggy==0.6 \
      PyGObject \
      pytest-cov \
      pytest==3.5.1
RUN git clone https://github.com/nmstate/nmstate.git && \
    cd nmstate && \
    pip install --upgrade .
COPY --from=builder /bin/state-handler /
ENTRYPOINT ["/state-handler"]
