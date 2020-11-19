#!/bin/bash -xe

# This script publish kubernetes-nmstate-handler by default at quay.io/nmstate
# organization to publish elsewhere export the following env vars
# IMAGE_REGISTRY
# IMAGE_REPO
# To run it just do proper docker login and automation/publish.sh

image_registry=${IMAGE_REGISTRY:-quay.io}
image_repo=${IMAGE_REPO:-nmstate}
branch=${PULL_BASE_REF:-$(git rev-parse --abbrev-ref HEAD)}
docs_source_branch=master
docs_container=${image_registry}/${image_repo}/kubernetes-nmstate-docs:$docs_source_branch

# Publish kubernets-nmstate containers
source automation/check-patch.setup.sh
(
    cd ${TMP_PROJECT_PATH}
    make \
        IMAGE_REGISTRY=${image_registry}  \
        IMAGE_REPO=${image_repo} \
        push-handler
        push-operator
)

# We don't have a versioned docs webpage so we only update it from master and
# not from release branches
if [ "$branch" == $docs_source_branch ]; then

    # Update kubernetes-nmstate documentation container
    docker build docs --build-arg BRANCH=$docs_source_branch -t $docs_container
    docker push $docs_container

    # To pass user/password from automations, idea is taken from [1]
    # [1] https://stackoverflow.com/questions/8536732/can-i-hold-git-credentials-in-environment-variables
    git config credential.helper '!f() { sleep 1; echo "username=${GITHUB_USER}"; echo "password=${GITHUB_TOKEN}"; }; f'

    # Update gh-pages branch with the generated documentation
    git checkout gh-pages --quiet
    git rm -r --quiet *
    docker run -v $(pwd):/gh-pages $docs_container bash -c "cp -r docs/build/kubernetes-nmstate/* /gh-pages"
    git add -A
    git commit -m "updated $(date +"%d.%m.%Y %H:%M:%S")"
    git push --quiet
    echo -e "\033[0;32mdemo updated $(date +"%d.%m.%Y %H:%M:%S")\033[0m"
    git checkout $docs_source_branch --quiet
fi
