FROM quay.io/centos/centos:stream8
ARG NMSTATE_COPR_REPO
ARG NM_COPR_REPO
RUN dnf install -b -y dnf-plugins-core && \
    dnf copr enable -y networkmanager/$NM_COPR_REPO && \
    dnf copr enable -y nmstate/$NMSTATE_COPR_REPO && \
    dnf copr enable -y nmstate/nispor && \
    dnf install -b -y nmstate iproute iputils && \
    dnf remove -y dnf-plugins-core && \
    dnf clean all

COPY build/_output/bin/manager /usr/local/bin/manager

ENTRYPOINT ["manager"]
