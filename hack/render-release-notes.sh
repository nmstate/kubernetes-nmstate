#!/bin/bash -e
script_dir=$(dirname "$(readlink -f "$0")")

old_version=$1
new_version=$2
release_notes=$(mktemp)

end() {
    rm $release_notes
}

trap end EXIT SIGINT SIGTERM SIGSTOP

$RELEASE_NOTES \
    --go-template go-template:$script_dir/release-notes.tmpl \
    --release-version $new_version \
    --required-author "" \
    --github-org nmstate \
    --github-repo kubernetes-nmstate \
    --start-rev $old_version \
    --end-rev $new_version \
    --output $release_notes > /dev/null 2>&1

cat $release_notes
