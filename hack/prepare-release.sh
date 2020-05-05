#!/bin/bash -e
script_dir=$(dirname "$(readlink -f "$0")")

version_type=$1
old_version=$(hack/version.sh)
new_version=$(hack/bump-version.sh $version_type)
release_notes=$(mktemp)

end() {
    rm $release_notes
}

trap end EXIT SIGINT SIGTERM SIGSTOP

$RELEASE_NOTES \
    --format go-template:$script_dir/release-notes.tmpl \
    --release-version $new_version \
    --required-author "" \
    --github-org nmstate \
    --github-repo kubernetes-nmstate \
    --start-rev $old_version \
    --end-rev master \
    --output $release_notes

cat << EOF > version/description
$new_version

TODO: Add description here

EOF

cat $release_notes >> version/description

cat << EOF >> version/description

\`\`\`
docker pull OPERATOR_IMAGE
\`\`\`
EOF

${EDITOR:-vi} version/description

git checkout -b release-$new_version
git commit -a -s -m "Release $new_version"
