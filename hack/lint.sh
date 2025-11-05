#!/bin/bash -xe
architecture=""
case $(uname -m) in
    x86_64)           architecture="amd64" ;;
    aarch64|arm64)    architecture="arm64" ;;
    *)
                      echo "Unsupported architecture: $(uname -m), must be (x86_64|aarch64|arm64)."
                      exit 1;
esac

path_combined="$(uname -s | tr '[:upper:]' '[:lower:]')-${architecture}"
golangci_lint_version=1.64.7
golangci_lint_url="https://github.com/golangci/golangci-lint/releases/download/v${golangci_lint_version}/golangci-lint-${golangci_lint_version}-${path_combined}.tar.gz"
golangci_cmd="/tmp/golangci-lint-${golangci_lint_version}-${path_combined}/golangci-lint"
if [ ! -f "${golangci_cmd}" ]; then
    curl -Lk $golangci_lint_url -o /tmp/golangci-lint-${golangci_lint_version}-${path_combined}.tar.gz
    tar -xvzf /tmp/golangci-lint-${golangci_lint_version}-${path_combined}.tar.gz -C /tmp
    chmod 755 ${golangci_cmd}
fi
${golangci_cmd} run --timeout 20m0s
(
	cd api
	${golangci_cmd} run --timeout 20m0s --config ../.golangci.yml
)
(
	cd automation/nmstate-latest-reporter
    ${golangci_cmd} run --timeout 20m0s --config ../../.golangci.yml --path-prefix=automation/nmstate-latest-reporter
)
