#!/bin/bash

set -euo pipefail

helm_bin=${HELM:-helm}
chart_version=${HELM_CHART_VERSION:-}
chart_app_version=${HELM_CHART_APP_VERSION:-}
chart_oci_repo=${HELM_CHART_OCI_REPO:-}

if [[ -z "${chart_version}" ]]; then
    echo "Error: HELM_CHART_VERSION is required (e.g. make HELM_CHART_VERSION=0.86.0 push-chart)" >&2
    exit 1
fi

if [[ -z "${chart_app_version}" ]]; then
    chart_app_version="v${chart_version}"
fi

if [[ -z "${chart_oci_repo}" ]]; then
    chart_oci_repo="oci://${IMAGE_REGISTRY:-quay.io}/${IMAGE_REPO:-nmstate}"
fi

# Both podman and docker may have performed the registry login, and each of
# them stores the credentials at a different location, so look them up at the
# usual places instead of assuming one of them.
helm_registry_config=${HELM_REGISTRY_CONFIG:-}
if [[ -z "${helm_registry_config}" ]]; then
    for candidate in \
        "${REGISTRY_AUTH_FILE:-}" \
        "${XDG_RUNTIME_DIR:-}/containers/auth.json" \
        "/run/containers/$(id -u)/auth.json" \
        "${HOME}/.config/containers/auth.json" \
        "${DOCKER_CONFIG:-${HOME}/.docker}/config.json"; do
        if [[ -n "${candidate}" && -f "${candidate}" ]]; then
            helm_registry_config="${candidate}"
            break
        fi
    done
fi

if [[ -z "${helm_registry_config}" ]]; then
    echo "Error: no registry credentials found, log in to ${chart_oci_repo#oci://} or set HELM_REGISTRY_CONFIG" >&2
    exit 1
fi

echo "Using registry credentials from ${helm_registry_config}"

"${helm_bin}" package charts/kubernetes-nmstate \
    --version "${chart_version}" \
    --app-version "${chart_app_version}" \
    --destination build/_output

"${helm_bin}" push "build/_output/kubernetes-nmstate-${chart_version}.tgz" \
    "${chart_oci_repo}" \
    --registry-config "${helm_registry_config}"
