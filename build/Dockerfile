FROM quay.io/centos/centos:stream8
ARG TARGETARCH

RUN env

COPY build/_output/bin/handler.manager.linux-$TARGETARCH /usr/local/bin/manager

# dnf install -b -y dnf-plugins-core && \
# FIXME: https://bugzilla.redhat.com/show_bug.cgi?id=2158151 \
RUN curl "https://people.redhat.com/fge/bz_2158151/nmstate_hotfix.repo" -o /etc/yum.repos.d/nmstate_hotfix.repo && \
    dnf install -b -y -x "*alpha*" -x "*beta*" nmstate && \
    dnf install -b -y iproute iputils && \
    dnf remove -y dnf-plugins-core && \
    dnf clean all

ENTRYPOINT ["manager"]
