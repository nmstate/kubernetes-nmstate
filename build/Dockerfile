FROM fedora:31

RUN sudo dnf install -y dnf-plugins-core && \
    sudo dnf copr enable -y nmstate/nmstate-0.2 && \
    sudo dnf install -y nmstate iproute iputils && \
    sudo dnf remove -y dnf-plugins-core && \
    sudo dnf clean all

# Looks like 35 is not enough we need 60 to propertly recover after reboot
# TODO: Remove this when we merge https://github.com/nmstate/nmstate/pull/555
RUN sed -i "s/MAINLOOP_TIMEOUT = 35/MAINLOOP_TIMEOUT = 60/g" /usr/lib/python3.7/site-packages/libnmstate/netapplier.py

COPY build/_output/bin/kubernetes-nmstate /usr/local/bin/kubernetes-nmstate
COPY build/bin /usr/local/bin

ENTRYPOINT ["kubernetes-nmstate"]
