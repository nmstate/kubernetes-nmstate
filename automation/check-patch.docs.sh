#!/bin/bash -xe

# Change url to point to google storage
url="https://storage.googleapis.com"
baseurl="kubevirt-prow/pr-logs/pull/nmstate_kubernetes-nmstate/${PULL_NUMBER}/pull-kubernetes-nmstate-docs/${BUILD_ID}/artifacts/gh-pages/"
sed -i "s#^url:.*#url: \"$url\"#" docs/_config.yaml
sed -i "s#^baseurl:.*#baseurl: \"$baseurl\"#" docs/_config.yaml

# Build the docs
docker build docs --build-arg BRANCH=${PULL_BASE_REF} -t kubernetes-nmstate-docs

# Copy the docs to the artifacts
mkdir -p $ARTIFACTS/gh-pages
docker run -v $ARTIFACTS:/artifacts kubernetes-nmstate-docs cp -r docs/build/$baseurl /artifacts

echo "kubernetes-nmstate preview URL: ${url}/${baseurl}index.html"
