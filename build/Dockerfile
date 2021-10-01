FROM rust as nmstate

RUN git clone https://github.com/ffmancera/nmstate -b go_bindings

#RUN --mount=type=cache,target=/usr/local/cargo/registry \
#	--mount=type=cache,target=/nmstate/rust/target \
RUN cd nmstate && \
    cd rust && \
    make

RUN ls -la /nmstate/rust/target/

FROM golang:1.16 AS manager 
ARG TARGETARCH

COPY go.* main.go . 
COPY controllers/ controllers/
COPY api/ api/
COPY pkg/ pkg/
COPY vendor vendor/
COPY --from=nmstate /nmstate/rust/target/release/libnmstate.so /usr/local/lib64/
COPY --from=nmstate /nmstate/rust/src/clib/nmstate.h.in /usr/local/include/nmstate.h

RUN GOFLAGS="-mod=vendor" GOPATH="" go build -o manager.linux-$TARGETARCH main.go

FROM centos:8
ARG NMSTATE_COPR_REPO
ARG NM_COPR_REPO
ARG TARGETARCH

RUN env

COPY --from=nmstate /nmstate/rust/target/release/ncl /usr/local/bin/
COPY --from=nmstate /nmstate/rust/target/release/libnmstate.so /usr/local/lib64/
COPY --from=manager /go/manager.linux-$TARGETARCH /usr/local/bin/manager

RUN dnf install -b -y iproute iputils && \
    dnf clean all

ENTRYPOINT ["manager"]
