FROM fedora:31

RUN sudo dnf install -y nmstate iproute iputils && \
    sudo dnf clean all

# TODO: Delete this line after we update nmstate to include the change
# https://github.com/nmstate/nmstate/commit/a4baaff50fef84d6d326977ff647bece1e917a26
RUN sed -i "s/run(timeout=20/run(timeout=60/g" /usr/lib/python3.7/site-packages/libnmstate/netapplier.py
# TODO: Delete this line after nmstate fixes https://github.com/nmstate/nmstate/issues/516
RUN sed -i "s/nmclient.NM.DeviceType.OVS_INTERFACE,/nmclient.NM.DeviceType.OVS_INTERFACE, nmclient.NM.DeviceType.BRIDGE,/" /usr/lib/python3.7/site-packages/libnmstate/nm/applier.py

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
