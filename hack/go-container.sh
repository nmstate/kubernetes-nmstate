#!/bin/sh
# Copyright 2019 The Kubernetes Authors.
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

# Simple posix sh reproducible go build container script with caching.
#
# Usage:
#  hack/go.sh go version
#  hack/go.sh go build -o /out/kind .
set -o nounset -o errexit

# ============================ SCRIPT SETTINGS =================================
# get the repo root for defaulting OUT_DIR and SOURCE_DIR
REPO_ROOT="${REPO_ROOT:-$(cd "$(dirname "$0")/.." && pwd)}"
# output directory, will be mounted to /out, defaults to /bin in REPO_ROOT
OUT_DIR="${OUT_DIR:-${REPO_ROOT}/build/_output/bin}"
# source directory, will be mounted to /src, defaults to current directory
SOURCE_DIR="${SOURCE_DIR:-$(pwd -P)}"
# default to disabling CGO for easier reproducible builds and cross compilation
export CGO_ENABLED="${CGO_ENABLED:-1}"
# the container image, by default a recent official golang image
GOIMAGE="${GOIMAGE:-golang:$(grep "^go " $REPO_ROOT/go.mod |awk '{print $2}')}"
# docker volume name, used as a go module / build cache
CACHE_VOLUME="${CACHE_VOLUME:-knmstate-build-cache}"
# allow overriding docker cli e.g podman
DOCKER="${DOCKER:-docker}"

export GO111MODULE=on

# ========================== END SCRIPT SETTINGS ===============================

# autodetects host GOOS and GOARCH and exports them if not set
detect_and_set_goos_goarch() {
  # detect GOOS equivalent if unset
  if [ -z "${GOOS:-}" ]; then
    case "$(uname -s)" in
      Darwin) export GOOS="darwin" ;;
      Linux) export GOOS="linux" ;;
      *) echo "Unknown host OS! '$(uname -s)'" exit 2 ;;
    esac
  fi

  # detect GOARCH equivalent if unset
  if [ -z "${GOARCH:-}" ]; then
    case "$(uname -m)" in
      x86_64) export GOARCH="amd64" ;;
      arm*)
        export GOARCH="arm"
        if [ "$(getconf LONG_BIT)" = "64" ]; then
          export GOARCH="arm64"
        fi
      ;;
      *) echo "Unknown host architecture! '$(uname -m)'" exit 2 ;;
    esac
  fi

  export GOOS GOARCH
}

# run $@ in a golang container with caching etc.
run_in_go_container() {
  "${DOCKER}" run \
    `# docker options: remove container on exit, run as the host user / group` \
      --privileged --network=host --rm --user "$(id -u):$(id -g)" \
    `# disable SELinux relabelling /src` \
      --security-opt label=disable \
      -v "${CACHE_VOLUME}:/go" -e XDG_CACHE_HOME=/go/cache \
    `# mount the output & source dir, set working directory to the source dir` \
      -v "${OUT_DIR}:/${OUT_DIR}" -v "${SOURCE_DIR}:${SOURCE_DIR}" -w "${SOURCE_DIR}" \
    `# pass through go settings: modules, proxy, cgo, OS / Arch` \
      -e HOME -e GOPATH -e GOROOT -e GOBIN -e GO111MODULE -e GOPROXY -e CGO_ENABLED -e GOFLAGS -e GOOS -e GOARCH \
    `# pass through proxy settings` \
      -e HTTP_PROXY -e HTTPS_PROXY -e NO_PROXY \
    `# TODO: Don't like it` \
      -e NODE_NAME -e INTERFACES_FILTER  \
    `# run the image with the args passed to this script` \
      "${GOIMAGE}" "$@"
}

mkdir -p "${OUT_DIR}"
"${DOCKER}" volume inspect "${CACHE_VOLUME}" >/dev/null || "${DOCKER}" volume create "${CACHE_VOLUME}" >/dev/null
detect_and_set_goos_goarch
run_in_go_container "$@"
