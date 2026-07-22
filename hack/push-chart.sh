#!/bin/bash

set -euo pipefail

helm_bin=${HELM:-helm}
chart_version=${CHART_VERSION:-}
chart_app_version=${CHART_APP_VERSION:-}
chart_oci_repo=${CHART_OCI_REPO:-}

if [[ -z "${chart_version}" ]]; then
    echo "Error: CHART_VERSION is required (e.g. make CHART_VERSION=0.86.0 push-chart)" >&2
    exit 1
fi

if [[ -z "${chart_app_version}" ]]; then
    chart_app_version="v${chart_version}"
fi

if [[ -z "${chart_oci_repo}" ]]; then
    chart_oci_repo="oci://${IMAGE_REGISTRY:-quay.io}/${IMAGE_REPO:-nmstate}"
fi

helm_registry_config=${HELM_REGISTRY_CONFIG:-}
if [[ -z "${helm_registry_config}" ]]; then
    if [[ -n "${XDG_RUNTIME_DIR:-}" && -f "${XDG_RUNTIME_DIR}/containers/auth.json" ]]; then
        helm_registry_config="${XDG_RUNTIME_DIR}/containers/auth.json"
    else
        helm_registry_config="${HOME}/.docker/config.json"
    fi
fi

"${helm_bin}" package charts/kubernetes-nmstate \
    --version "${chart_version}" \
    --app-version "${chart_app_version}" \
    --destination build/_output

"${helm_bin}" push "build/_output/kubernetes-nmstate-${chart_version}.tgz" \
    "${chart_oci_repo}" \
    --registry-config "${helm_registry_config}"
