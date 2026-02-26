#!/bin/bash -xe

IMAGE_BUILDER=${IMAGE_BUILDER:-$(./hack/detect_cri.sh)}

url="https://storage.googleapis.com"
baseurl="/${PULL_NUMBER}/pull-kubernetes-nmstate-docs/${BUILD_ID}/artifacts/gh-pages/"
preview_baseurl="${url}/kubevirt-prow/pr-logs/pull/nmstate_kubernetes-nmstate${baseurl}"

# Build docs, check links, then rebuild with preview base URL
${IMAGE_BUILDER} run -v $(pwd)/docs:/docs/ -w /docs \
    -e GOBIN=/usr/local/bin \
    docker.io/hugomods/hugo:exts \
    sh -c "npm install && command -v htmltest || go install github.com/wjdp/htmltest@latest && hugo --baseURL / --destination build/ && htmltest && hugo --baseURL '${preview_baseurl}' --destination build/"

# Copy the docs to the artifacts
mkdir -p $ARTIFACTS/gh-pages
rsync -rt --links docs/build/* $ARTIFACTS/gh-pages

echo "kubernetes-nmstate preview URL: ${preview_baseurl}index.html"
