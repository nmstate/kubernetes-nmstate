ARG FROM=quay.io/centos/centos:stream8
FROM --platform=linux/amd64 ${FROM} AS nmstate

RUN 
RUN curl https://sh.rustup.rs -sSf -o /tmp/rustup-init.sh && \
    sh /tmp/rustup-init.sh -y && \
    dnf install -y git make gcc

RUN git clone https://github.com/cathay4t/nmstate -b new_release

WORKDIR nmstate

RUN --mount=type=cache,target=/root/.cargo/registry \
    --mount=type=cache,target=/nmstate/target \
    source $HOME/.cargo/env && make install

ARG FROM=quay.io/centos/centos:stream8
FROM --platform=linux/amd64 ${FROM} AS build

ARG TARGETARCH

RUN mkdir /workdir

COPY go.mod /workdir

WORKDIR /workdir

COPY --from=nmstate /usr/local/lib/libnmstate* /usr/local/lib/
COPY --from=nmstate /usr/local/include/nmstate* /usr/local/include

RUN dnf install -b -y dnf-plugins-core && \
    dnf install -b -y make golang-$(sed -En 's/^go +(.*+)$/\1/p' go.mod).*

COPY . .

RUN GOOS=linux GOARCH=${TARGETARCH} go build -o /manager ./cmd/handler

ARG FROM=quay.io/centos/centos:stream8
FROM ${FROM}

COPY --from=nmstate /usr/local/lib/libnmstate* /usr/local/lib/
COPY --from=build /manager /usr/local/bin/manager
COPY --from=build /workdir/build/install-nmstate.${NMSTATE_SOURCE}.sh install-nmstate.sh

RUN dnf install -b -y iproute iputils && \
    dnf clean all

ENTRYPOINT ["manager"]
