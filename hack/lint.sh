#!/bin/bash -e
golangci_lint_version=v1.42.1
if [ ! -f $(go env GOPATH)/bin/golangci-lint ]; then
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin $golangci_lint_version
fi
golangci-lint run --timeout 5m0s
(
	cd api
	golangci-lint run --timeout 5m0s --config ../.golangci.yml
)
