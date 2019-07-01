#!/bin/bash -xe

git tag $TAG
git push upstream --tags

$GITHUB_RELEASE release -u nmstate -r kubernetes-nmstate \
    --tag $TAG \
	--name $TAG \
    --description "$(cat $DESCRIPTION)"

for resource in "$@" ;do
    $GITHUB_RELEASE upload -u nmstate -r kubernetes-nmstate \
        --name $(basename $resource) \
	    --tag $TAG \
		--file $resource
done
