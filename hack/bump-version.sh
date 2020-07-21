#!/bin/bash -e

expected_types="(major|minor|patch)"
current_type=$1

bump() {
    version=$(hack/versions.sh -1)
    version_part=$(echo $version |sed $1)
    version_part=$((++version_part))
    version=$(echo $version | sed $2 | sed "s/version_part/$version_part/g")
    echo v$version
}

bump_major() {
   bump "s/^v\(.*\)[.].*[.].*$/\1/g" "s/^v.*$/version_part.0.0/g"
}

bump_minor() {
    bump "s/^v.*[.]\(.*\)[.].*$/\1/g" "s/^v\(.*\)[.].*[.].*$/\1.version_part.0/g"
}

bump_patch() {
    bump "s/^v.*[.].*[.]\(.*\)$/\1/g" "s/^v\(.*\)[.]\(.*\)[.].*$/\1.\2.version_part/g"
}

if [[ ! $current_type =~ $expected_types ]]; then
    echo "Usage: $0 $expected_types"
    exit 1
fi

bump_$current_type
