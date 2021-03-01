FROM centos:8
ARG NMSTATE_COPR_REPO
ARG NM_COPR_REPO
RUN dnf install -b -y dnf-plugins-core && \
    dnf copr enable -y networkmanager/$NM_COPR_REPO && \
    dnf copr enable -y packit/nmstate-nmstate-1530 && \
    dnf copr enable -y nmstate/nispor && \
    dnf install -b -y nmstate iproute iputils && \
    dnf remove -y dnf-plugins-core && \
    dnf clean all

COPY build/_output/bin/manager /usr/local/bin/manager
COPY build/bin /usr/local/bin

ENTRYPOINT ["manager"]
