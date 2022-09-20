#!/usr/bin/env bash

# This script helps with rebasing the openshift repo to upstream. 
# It follows the procedure from https://github.com/openshift/kubernetes-nmstate/pull/298. 
# Anyhow the user should check that all "UPSTREAM: <carry>" commits got carried,
# especially in case some were merged between creating and merging the last
# rebase.

set -e

git remote update

upstream_remote=$(git remote -v | grep "https://github.com/nmstate/kubernetes-nmstate (fetch)" | awk '{print $1}')
downstream_remote=$(git remote -v | grep "https://github.com/openshift/kubernetes-nmstate (fetch)" | awk '{print $1}')

read -p "Source branch from upstream (usually main): " upstream_source_branch
if ! git show-ref --quiet ${upstream_remote}/${upstream_source_branch}; then
    echo "Branch ${upstream_source_branch} does not exist in ${upstream_remote}"
    exit 1
fi

read -p "Target branch in downstream (usually master): " downstream_target_branch
if ! git show-ref --quiet ${downstream_remote}/${downstream_target_branch}; then
    echo "Branch ${downstream_target_branch} does not exist in ${downstream_remote}"
    exit 1
fi

read -p "CommitId of latest merge commit in ${downstream_remote}/${downstream_target_branch}: " last_merge_commit
if ! git log ${downstream_remote}/${downstream_target_branch} | grep -q ${last_merge_commit}; then
    echo "commit ${last_merge_commit} not found in ${downstream_remote}/${downstream_target_branch}. Aborting..."
    exit 1   
fi

git branch -D merge-tmp || true # make sure old merge-tmp branch does not exist
git checkout ${upstream_remote}/${upstream_source_branch}
git checkout -b merge-tmp # create a branch to do our merge work from
git checkout ${downstream_remote}/${downstream_target_branch} # we want to be at the tip of the openshift master branch when we run the next command

merge_commit_msg="merge nmstate/${upstream_source_branch} $(date +%Y-%m-%d)"
merge_commit=$(echo "${merge_commit_msg}" | git commit-tree merge-tmp^{tree} -p HEAD -p merge-tmp -F -)

merge_branch=merge-${upstream_source_branch}-$(date +%Y-%m-%d)-${merge_commit:0:8}
git branch -D ${merge_branch} || true # make sure old merge branch does not exist
git branch ${merge_branch} ${merge_commit} # create a new branch for the cherry-pick work
git checkout ${merge_branch}

echo "Cherry-picking commits since ${last_merge_commit}..."
for commit in $(git --no-pager log --oneline --reverse --no-merges ${last_merge_commit}..${downstream_remote}/${downstream_target_branch} | awk '{print $1}'); do
    echo "cherry-picking "$(git --no-pager log --format=%s -n 1 ${commit} ${downstream_remote}/${downstream_target_branch})

    if ! git cherry-pick -x $commit; then
        echo "Error on cherry-picking happened. Maybe a merge commit you have to resolve and fix."
        read -p "Press any key to continue when issue is fixed..."
    fi
done

echo "Cherry-picking done"
echo
echo

echo "Cherry-picked commit since (${last_merge_commit})"
echo "New merge commit is \"${merge_commit_msg} (${merge_commit})\""
echo
echo "Cherry-picked/carried commits:"
for commit in $(git --no-pager log --oneline --reverse --no-merges ${merge_commit}..${merge_branch} | awk '{print $1}'); do
    commit_msg=$(git --no-pager log --format=%s -n 1 ${commit})
    echo "${commit_msg} (${commit})"
done

echo "Please make sure the commits have the \"UPSTREAM: <carry>:\" prefix and check if you can squash commits" 
echo "Please make also sure all commits were carried. Maybe even check for some merged commits before the last merge commit (${last_merge_commit}) and cherry-pick them manually if needed."
