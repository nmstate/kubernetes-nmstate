FROM centos:7

RUN yum -y install iproute NetworkManager-libnm git cairo* epel-release && \
    yum -y install python2-pip gcc python-devel gobject-introspection-devel && \
    yum clean all

RUN pip install -U \
      pip \
      setuptools \
      pbr \
      tox \
      pluggy==0.6 \
      PyGObject \
      pytest-cov \
      pytest==3.5.1

RUN git clone https://github.com/nmstate/nmstate.git && \
    cd nmstate && \
    pip install --upgrade .

COPY ./state-controller /

ENTRYPOINT ["/state-controller"]
