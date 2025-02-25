#!/bin/bash -xe
golangci_lint_version=1.63.4
golangci_lint_url=https://github.com/golangci/golangci-lint/releases/download/v${golangci_lint_version}/golangci-lint-${golangci_lint_version}-linux-amd64.tar.gz
golangci_cmd=/tmp/golangci-lint-${golangci_lint_version}-linux-amd64/golangci-lint
if [ ! -f "${golangci_cmd}" ]; then
    curl -Lk $golangci_lint_url -o /tmp/golangci-lint-${golangci_lint_version}-linux-amd64.tar.gz
    tar -xvzf /tmp/golangci-lint-${golangci_lint_version}-linux-amd64.tar.gz -C /tmp
    chmod 755 ${golangci_cmd}
fi
${golangci_cmd} run --timeout 20m0s
(
	cd api
	${golangci_cmd} run --timeout 20m0s --config ../.golangci.yml
)
