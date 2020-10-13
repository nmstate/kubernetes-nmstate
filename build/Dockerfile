FROM centos:8

RUN dnf install -b -y dnf-plugins-core && \
    dnf copr enable -y networkmanager/NetworkManager-1.22 && \
    dnf copr enable -y nmstate/nmstate-0.2 && \
    dnf install -b -y nmstate iproute iputils && \
    dnf remove -y dnf-plugins-core && \
    dnf clean all

COPY build/_output/bin/manager /usr/local/bin/manager
COPY build/bin /usr/local/bin

ENTRYPOINT ["manager"]
