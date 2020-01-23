#!/bin/bash -e
version_type=$1
old_version=$(hack/version.sh)
new_version=$(hack/bump-version.sh $version_type)
commits=$(git log --pretty=format:"* %s" $old_version..HEAD)


cat << EOF > version/description
$new_version

TODO: Add description here


TODO: keep at every category the
      commits that make sense

Features:
$commits

Bugs:
$commits

Docs:
$commits

\`\`\`
docker pull HANDLER_IMAGE
\`\`\`
EOF

${EDITOR:-vi} version/description

git checkout -b release-$new_version
git commit -a -s -m "Release $new_version"
