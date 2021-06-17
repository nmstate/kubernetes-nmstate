#!/bin/bash -e
script_dir=$(dirname "$(readlink -f "$0")")

old_version=$1
new_version=$2
release_notes=$(mktemp)

end() {
    rm $release_notes
}

trap end EXIT SIGINT SIGTERM SIGSTOP

$GO run k8s.io/release/cmd/release-notes \
    --list-v2 \
    --go-template go-template:$script_dir/release-notes.tmpl \
    --required-author "" \
    --org nmstate \
    --repo kubernetes-nmstate \
    --start-rev $old_version \
    --end-rev $new_version \
    --output $release_notes > release-notes.log 2>&1

cat $release_notes
