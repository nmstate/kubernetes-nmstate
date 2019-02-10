#!/usr/bin/env bash
# Copyright 2019 The nmstate Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Configurable environment variables:
# MANIFESTS_SOURCE
# MANIFESTS_DESTINATION
# NAMESPACE
# IMAGE_REGISTRY
# IMAGE_TAG
# PULL_POLICY
# STATE_CLIENT_IMAGE
# STATE_CONTROLLER_IMAGE

set -euo pipefail

templates="$(find "${MANIFESTS_SOURCE}" -name "*.in" -type f)"

(cd tools/manifest-generator/ && go fmt && go build -o manifest-generator)

mkdir -p "${MANIFESTS_DESTINATION}/"

for tmpl in ${templates}; do
    tmpl=$(readlink -f "${tmpl}")
    out_file=$(basename -s .in "${tmpl}")
    (tools/manifest-generator/manifest-generator -template="${tmpl}" \
        -namespace="${NAMESPACE}" \
        -image-registry="${IMAGE_REGISTRY}" \
        -image-tag="${IMAGE_TAG}" \
        -pull-policy="${PULL_POLICY}" \
        -state-client-image="${STATE_CLIENT_IMAGE}" \
        -state-controller-image="${STATE_CONTROLLER_IMAGE}"
    ) > "${MANIFESTS_DESTINATION}/${out_file}"
done

# Remove empty lines at the end of files which are added by go templating
find ${MANIFESTS_DESTINATION}/ -type f -exec sed -i {} -e '${/^$/d;}' \;
