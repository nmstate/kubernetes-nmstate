#!/bin/bash

set -euo pipefail

# Focused render test for the kubernetes-nmstate Helm chart: string-valued
# fields (names, namespaces, images, pull policies, env values) must survive
# YAML-ambiguous but valid DNS-label inputs such as "null", "yes" or "123".
# Without explicit quoting these decode as null/boolean/integer values and
# the resulting objects fail Kubernetes schema validation.

HELM=${HELM:-helm}
CHART_DIR=${CHART_DIR:-charts/kubernetes-nmstate}

rendered=$(${HELM} template nmstate "${CHART_DIR}" \
    --namespace null \
    --set createNamespace=true \
    --set handler.namespace=yes \
    --set monitoring.namespace=123)

fail() {
    echo "helm render test failed: $1" >&2
    echo "${rendered}" >&2
    exit 1
}

echo "${rendered}" | grep -q 'namespace: "null"' \
    || fail 'expected metadata.namespace to render as the string "null"'
echo "${rendered}" | grep -q 'value: "yes"' \
    || fail 'expected HANDLER_NAMESPACE env value to render as the string "yes"'
echo "${rendered}" | grep -q 'value: "123"' \
    || fail 'expected MONITORING_NAMESPACE env value to render as the string "123"'

# No string-valued field may render one of the ambiguous inputs unquoted
if echo "${rendered}" | grep -E '^\s*(name|namespace|value):\s*(null|yes|123)\s*$'; then
    fail 'found unquoted YAML-ambiguous value in rendered output'
fi

echo "helm render test passed"
