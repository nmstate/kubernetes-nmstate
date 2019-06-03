FROM centos:7
RUN yum-config-manager --add-repo https://copr.fedorainfracloud.org/coprs/nmstate/nmstate-el7/repo/epel-7/nmstate-nmstate-el7-epel-7.repo && \
    yum -y install epel-release && \
    yum -y update && \
    yum -y install nmstate && \
    yum clean all
CMD exec /bin/bash -c "trap : TERM INT; sleep infinity & wait"
