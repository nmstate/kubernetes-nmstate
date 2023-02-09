#!/bin/bash -xe
golangci_lint_version=v1.50.1
GOFLAGS=-mod=mod go run github.com/golangci/golangci-lint/cmd/golangci-lint@$golangci_lint_version run --timeout 20m0s
(
	cd api
	GOFLAGS=-mod=mod go run github.com/golangci/golangci-lint/cmd/golangci-lint@$golangci_lint_version run --timeout 20m0s --config ../.golangci.yml
)
