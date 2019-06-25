FROM centos:7

RUN yum -y install epel-release && \
    yum -y update && \
    yum -y install \
        https://kojipkgs.fedoraproject.org/packages/nmstate/0.0.6/2.el7/noarch/python2-libnmstate-0.0.6-2.el7.noarch.rpm \
        https://kojipkgs.fedoraproject.org/packages/nmstate/0.0.6/2.el7/noarch/nmstate-0.0.6-2.el7.noarch.rpm \
    yum clean all

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
