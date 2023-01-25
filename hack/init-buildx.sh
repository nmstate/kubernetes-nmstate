#!/usr/bin/env bash
# Copyright 2020 The Kubernetes Authors.
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

set -o errexit -o nounset -o pipefail

export DOCKER_CLI_EXPERIMENTAL=enabled

# If there is no buildx let's install it
if ! docker buildx > /dev/null 2>&1; then
    mkdir -p ~/.docker/cli-plugins
    curl -L https://github.com/docker/buildx/releases/download/v0.6.3/buildx-v0.6.3.linux-amd64 --output ~/.docker/cli-plugins/docker-buildx
    chmod a+x ~/.docker/cli-plugins/docker-buildx
fi


# We can skip setup if the current builder already has multi-arch
# AND if it isn't the docker driver, which doesn't work
current_builder="$(docker buildx inspect)"
# linux/amd64, linux/arm64, linux/riscv64, linux/ppc64le, linux/s390x, linux/386, linux/arm/v7, linux/arm/v6
if ! grep -q "^Driver: docker$"  <<<"${current_builder}" && \
     grep -q "linux/amd64" <<<"${current_builder}" && \
     grep -q "linux/arm64" <<<"${current_builder}"; then
  exit 0
fi


if ! docker buildx ls | grep -q kubernetes-nmstate-builder; then
    docker buildx create --driver-opt network=host --use --name=kubernetes-nmstate-builder
fi
