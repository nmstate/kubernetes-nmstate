FROM registry.access.redhat.com/ubi7/ubi-minimal:latest

# Cannot change the binary to handler-manager since the name
# is taken from the directory name [1]
# [1] https://github.com/operator-framework/operator-sdk/blob/master/cmd/operator-sdk/build/cmd.go#L103

ENV OPERATOR=/usr/local/bin/kubernetes-nmstate \
    USER_UID=1001 \
    USER_NAME=kubernetes-nmstate

# install operator binary
COPY build/_output/bin/kubernetes-nmstate ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
