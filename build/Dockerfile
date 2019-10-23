FROM fedora:31

RUN sudo dnf install -y dnf-plugins-core && \
    sudo dnf copr enable -y nmstate/nmstate-git-fedora && \
    sudo dnf install -y nmstate-0.0.8-0.20191022.669git5701bbe.fc31.noarch iproute iputils && \
    sudo dnf remove -y dnf-plugins-core && \
    sudo dnf clean all

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
