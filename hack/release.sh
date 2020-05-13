#!/bin/bash

set -xe

tag=$(hack/version.sh)

function upload() {
    resource=$1
    $GITHUB_RELEASE upload -u nmstate -r kubernetes-nmstate \
        --name $(basename $resource) \
	    --tag $tag \
		--file $resource
}

function create_github_release() {
    # Create the release
    $GITHUB_RELEASE release -u nmstate -r kubernetes-nmstate \
        --tag $tag \
        --name $tag \
        --description "$(cat version/description)"


    # Upload operator CRDs
    for manifest in $(ls deploy/crds/nmstate.io_*nmstate*); do
        upload $manifest
    done

    # Upload operator manifests
    for manifest in $(find $MANIFESTS_DIR -type f); do
        upload $manifest
    done
}

make OPERATOR_IMAGE_SUFFIX=:$tag HANDLER_IMAGE_SUFFIX=:$tag \
    manifests \
    push-handler \
    push-operator

# Tag master
git tag $tag
git push https://github.com/nmstate/kubernetes-nmstate $tag


create_github_release
