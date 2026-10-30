#!/bin/bash

set -euo pipefail

helm_bin=${HELM:-helm}
chart_version=${HELM_CHART_VERSION:-}
chart_app_version=${HELM_CHART_APP_VERSION:-}
chart_oci_repo=${HELM_CHART_OCI_REPO:-}
sign_key=${HELM_CHART_SIGN_KEY:-}
sign_keyring=${HELM_CHART_SIGN_KEYRING:-}
sign_passphrase_file=${HELM_CHART_SIGN_PASSPHRASE_FILE:-}

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

sign_args=()
if [[ -n "${sign_key}${sign_keyring}${sign_passphrase_file}" ]]; then
    if [[ -z "${sign_key}" || -z "${sign_keyring}" ]]; then
        echo "Error: HELM_CHART_SIGN_KEY and HELM_CHART_SIGN_KEYRING must both be set to sign a chart" >&2
        exit 1
    fi
    if [[ ! -f "${sign_keyring}" || ! -r "${sign_keyring}" ]]; then
        echo "Error: HELM_CHART_SIGN_KEYRING must point to a readable OpenPGP keyring" >&2
        exit 1
    fi
    # Helm requires at least one line in the passphrase file. Supply an
    # empty passphrase for unencrypted keys without prompting in CI.
    if [[ -z "${sign_passphrase_file}" ]]; then
        temporary_passphrase_file=$(mktemp)
        trap 'rm -f -- "${temporary_passphrase_file}"' EXIT
        printf '\n' > "${temporary_passphrase_file}"
        sign_passphrase_file=${temporary_passphrase_file}
    fi
    if [[ ! -r "${sign_passphrase_file}" ]]; then
        echo "Error: HELM_CHART_SIGN_PASSPHRASE_FILE must point to a readable file" >&2
        exit 1
    fi
    sign_args=(--sign --key "${sign_key}" --keyring "${sign_keyring}"
        --passphrase-file "${sign_passphrase_file}")
fi

helm_registry_config=${HELM_REGISTRY_CONFIG:-}
if [[ -z "${helm_registry_config}" ]]; then
    if [[ -n "${XDG_RUNTIME_DIR:-}" && -f "${XDG_RUNTIME_DIR}/containers/auth.json" ]]; then
        helm_registry_config="${XDG_RUNTIME_DIR}/containers/auth.json"
    else
        helm_registry_config="${HOME}/.docker/config.json"
    fi
fi

chart_archive="build/_output/kubernetes-nmstate-${chart_version}.tgz"
# Helm automatically pushes an adjacent .prov file. Remove any signature
# from an earlier package of this version before creating a new archive.
rm -f -- "${chart_archive}.prov"

"${helm_bin}" package charts/kubernetes-nmstate \
    --version "${chart_version}" \
    --app-version "${chart_app_version}" \
    --destination build/_output \
    "${sign_args[@]}"

if [[ ${#sign_args[@]} -gt 0 ]]; then
    "${helm_bin}" verify "${chart_archive}" --keyring "${sign_keyring}"
fi

"${helm_bin}" push "${chart_archive}" \
    "${chart_oci_repo}" \
    --registry-config "${helm_registry_config}"
