FROM centos:7

RUN yum -y install epel-release && \
    yum -y update && \
    yum -y install \
        https://kojipkgs.fedoraproject.org/packages/nmstate/0.0.6/2.el7/noarch/python2-libnmstate-0.0.6-2.el7.noarch.rpm \
        https://kojipkgs.fedoraproject.org/packages/nmstate/0.0.6/2.el7/noarch/nmstate-0.0.6-2.el7.noarch.rpm \
    yum clean all
CMD exec /bin/bash -c "trap : TERM INT; sleep infinity & wait"
