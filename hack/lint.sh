#!/bin/bash -xe
golangci_lint_version=v1.42.1
if [ ! -f $(go env GOPATH)/bin/golangci-lint ]; then
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin $golangci_lint_version
fi
golangci-lint run --timeout 10m0s
(
	cd api
	GOFLAGS=-mod=mod golangci-lint run --timeout 10m0s --config ../.golangci.yml
)
