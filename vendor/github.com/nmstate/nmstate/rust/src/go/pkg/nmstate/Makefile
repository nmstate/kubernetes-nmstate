# This Makefile is handling the go build, install, test and clean. It is needed
# to avoid isues with the GOROOT and GOPATH.

all: build check install

build:
	go build;

install: build
	go install;

check:
	go test;

clean:
	go clean;
