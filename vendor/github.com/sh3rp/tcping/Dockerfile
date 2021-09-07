FROM golang:latest

WORKDIR /go/src/github.com/sh3rp/tcping
COPY . .

RUN go-wrapper download
RUN go-wrapper install

ENV GOBIN /go/bin
