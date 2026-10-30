#!/bin/bash

set -euo pipefail

# Exercise real Helm packaging and OpenPGP verification, intercepting only
# the OCI push so this test never publishes a chart.
repo_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
real_helm=$(command -v "${HELM:-helm}")
if [[ "${real_helm}" != /* ]]; then
    real_helm="${PWD}/${real_helm}"
fi
command -v gpg >/dev/null

umask 077
tmpdir=$(mktemp -d)
cleanup() {
    gpgconf --homedir "${tmpdir}/gnupg" --kill gpg-agent >/dev/null 2>&1 || true
    rm -rf "${tmpdir}"
}
trap cleanup EXIT

mkdir -p "${tmpdir}/gnupg" "${tmpdir}/work/charts" "${tmpdir}/work/build/_output"
cp -RL "${repo_dir}/charts/kubernetes-nmstate" "${tmpdir}/work/charts/"

export REAL_HELM="${real_helm}"
export PUSH_ARTIFACTS="${tmpdir}/pushed"
export VERIFY_MARKER="${tmpdir}/verified"
export GNUPGHOME="${tmpdir}/gnupg"
export HELM="${tmpdir}/helm"
export HELM_CHART_VERSION=0.0.0
export HELM_CHART_APP_VERSION=v0.0.0
export HELM_CHART_OCI_REPO=oci://example.invalid/charts
export HELM_REGISTRY_CONFIG="${tmpdir}/registry.json"
export HELM_CHART_SIGN_KEY=
export HELM_CHART_SIGN_KEYRING=
export HELM_CHART_SIGN_PASSPHRASE_FILE=
export CORRUPT_ARCHIVE=false
printf '{}\n' > "${HELM_REGISTRY_CONFIG}"

cat > "${HELM}" <<'EOF'
#!/bin/bash
set -euo pipefail
case "$1" in
    package)
        "${REAL_HELM}" "$@"
        if [[ "${CORRUPT_ARCHIVE}" == true ]]; then
            printf 'tampered\n' >> build/_output/kubernetes-nmstate-0.0.0.tgz
        fi
        ;;
    verify)
        "${REAL_HELM}" "$@"
        touch "${VERIFY_MARKER}"
        ;;
    push)
        [[ "$3" == oci://example.invalid/charts ]]
        mkdir -p "${PUSH_ARTIFACTS}"
        cp "$2" "${PUSH_ARTIFACTS}/"
        if [[ -f "$2.prov" ]]; then
            [[ -f "${VERIFY_MARKER}" ]]
            cp "$2.prov" "${PUSH_ARTIFACTS}/"
        fi
        ;;
    *)
        echo "Unexpected Helm command: $1" >&2
        exit 1
        ;;
esac
EOF
chmod +x "${HELM}"

fail() {
    echo "chart signing test failed: $1" >&2
    exit 1
}

reset_push() {
    rm -rf "${PUSH_ARTIFACTS}"
    rm -f "${VERIFY_MARKER}"
}

expect_failure() {
    local description=$1
    shift
    reset_push
    if "$@" > "${tmpdir}/failure.log" 2>&1; then
        cat "${tmpdir}/failure.log" >&2
        fail "${description}: expected a failure"
    fi
    [[ ! -d "${PUSH_ARTIFACTS}" ]] || fail "${description}: attempted a push"
    echo "Passed: ${description}"
}

cd "${tmpdir}/work"
script="${repo_dir}/hack/push-chart.sh"
archive=kubernetes-nmstate-0.0.0.tgz

# A stale signature from a previous build must not accompany an unsigned chart.
printf 'stale provenance\n' > "build/_output/${archive}.prov"
bash "${script}"
[[ -f "${PUSH_ARTIFACTS}/${archive}" ]] || fail 'unsigned chart was not pushed'
[[ ! -f "${PUSH_ARTIFACTS}/${archive}.prov" ]] || fail 'stale provenance was pushed'

identity='Helm chart test <chart-test@example.invalid>'
keyring="${tmpdir}/secret.gpg"
passphrase_file="${tmpdir}/passphrase"
printf 'test-only-passphrase\n' > "${passphrase_file}"
gpg --batch --pinentry-mode loopback --passphrase-file "${passphrase_file}" \
    --quick-generate-key "${identity}" rsa2048 sign 1d
gpg --batch --pinentry-mode loopback --passphrase-file "${passphrase_file}" \
    --export-secret-keys "${identity}" > "${keyring}"
gpg --batch --export "${identity}" > "${tmpdir}/public.gpg"

expect_failure 'missing signing keyring' env HELM_CHART_SIGN_KEY="${identity}" bash "${script}"
expect_failure 'missing signing identity' env HELM_CHART_SIGN_KEYRING="${keyring}" bash "${script}"
expect_failure 'passphrase without signing key' env \
    HELM_CHART_SIGN_PASSPHRASE_FILE="${passphrase_file}" bash "${script}"
expect_failure 'unreadable signing keyring' env HELM_CHART_SIGN_KEY="${identity}" \
    HELM_CHART_SIGN_KEYRING="${tmpdir}/missing.gpg" bash "${script}"

export HELM_CHART_SIGN_KEY="${identity}"
export HELM_CHART_SIGN_KEYRING="${keyring}"
export HELM_CHART_SIGN_PASSPHRASE_FILE="${passphrase_file}"
reset_push
bash "${script}"
[[ -f "${PUSH_ARTIFACTS}/${archive}.prov" ]] || fail 'signed chart provenance was not pushed'
"${real_helm}" verify "${PUSH_ARTIFACTS}/${archive}" --keyring "${tmpdir}/public.gpg"

expect_failure 'tampered chart fails verification before push' env CORRUPT_ARCHIVE=true bash "${script}"
printf 'incorrect-passphrase\n' > "${tmpdir}/wrong-passphrase"
expect_failure 'incorrect passphrase' env \
    HELM_CHART_SIGN_PASSPHRASE_FILE="${tmpdir}/wrong-passphrase" bash "${script}"
expect_failure 'unreadable passphrase file' env \
    HELM_CHART_SIGN_PASSPHRASE_FILE="${tmpdir}/missing-passphrase" bash "${script}"

# Unencrypted keys must work without interactive input or a passphrase file.
identity='Unencrypted Helm test <unencrypted-test@example.invalid>'
gpg --batch --pinentry-mode loopback --passphrase '' \
    --quick-generate-key "${identity}" rsa2048 sign 1d
gpg --batch --export-secret-keys "${identity}" > "${tmpdir}/unencrypted.gpg"
reset_push
env HELM_CHART_SIGN_KEY="${identity}" HELM_CHART_SIGN_KEYRING="${tmpdir}/unencrypted.gpg" \
    HELM_CHART_SIGN_PASSPHRASE_FILE= bash "${script}"
[[ -f "${PUSH_ARTIFACTS}/${archive}.prov" ]] || fail 'unencrypted signing key did not produce provenance'

echo 'Helm chart signing tests passed'
