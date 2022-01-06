FROM quay.io/centos/centos:stream8
ARG TARGETARCH

RUN env

COPY build/_output/bin/handler.manager.linux-$TARGETARCH /usr/local/bin/manager

RUN dnf install -b -y dnf-plugins-core && \
    dnf install -b -y -x "*alpha*" -x "*beta*" nmstate-1.0.2 && \
    dnf install -b -y iproute iputils && \
    dnf remove -y dnf-plugins-core && \
    dnf clean all

ENTRYPOINT ["manager"]
