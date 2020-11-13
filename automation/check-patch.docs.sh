#!/bin/bash -xe

# Change url to point to google storage
url="https://storage.googleapis.com"
baseurl="kubevirt-prow/pr-logs/pull/nmstate_kubernetes-nmstate/${PULL_NUMBER}/pull-kubernetes-nmstate-docs/${BUILD_ID}/artifacts/gh-pages/"
sed -i "s#^url:.*#url: \"$url\"#" docs/_config.yaml
sed -i "s#^baseurl:.*#baseurl: \"$baseurl\"#" docs/_config.yaml


docker run -v $(pwd)/docs:/docs/ ruby make -C docs install check

# Copy the docs to the artifacts
mkdir -p $ARTIFACTS/gh-pages
rsync -rt --links docs/build/$baseurl $ARTIFACTS/gh-pages

echo "kubernetes-nmstate preview URL: ${url}/${baseurl}index.html"
