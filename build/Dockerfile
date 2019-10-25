FROM fedora:31

RUN sudo dnf install -y dnf-plugins-core && \
    sudo dnf copr enable -y nmstate/nmstate-git-fedora && \
    sudo dnf install -y nmstate-0.0.8-0.20191022.669git5701bbe.fc31.noarch iproute iputils && \
    sudo dnf remove -y dnf-plugins-core && \
    sudo dnf clean all

# Looks like 35 is not enough we need 60 to propertly recover after reboot
# TODO: Remove this when we merge https://github.com/nmstate/nmstate/pull/555
RUN sed -i "s/MAINLOOP_TIMEOUT = 35/MAINLOOP_TIMEOUT = 60/g" /usr/lib/python3.7/site-packages/libnmstate/netapplier.py

# Cannot change the binary to nmstate-handler since the name
# is taken from the directory name [1]
# [1] https://github.com/operator-framework/operator-sdk/blob/master/cmd/operator-sdk/build/cmd.go#L103
ENV OPERATOR=/usr/local/bin/kubernetes-nmstate \
    USER_UID=0 \
    USER_NAME=root

# install operator binary
COPY build/_output/bin/kubernetes-nmstate ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
