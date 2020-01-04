#!/usr/bin/env bash
#
# Check for trailing whitespaces in all tracked files.
#
# Usage:
# hack/whitespace.sh check # Check for trailing whitespaces
# hack/whitespace.sh format # Drop trailing whitespaces

set -e

function format() {
    git ls-files | grep -v "^vendor/" | xargs sed --follow-symlinks -i 's/[[:space:]]*$//'
}

function check() {
    invalid_files=$(git ls-files | grep -v "^vendor/" | xargs egrep -Hn " +$" || true)
    if [[ $invalid_files ]]; then
        echo 'Found trailing whitespaces. Please remove trailing whitespaces using `make format`:'
        echo "$invalid_files"
        return 1
    fi
}

if [ "$1" == "format" ]; then
    format
elif [ "$1" == "check" ]; then
    check
else
    echo "Please provide an argument [format|check]"
    exit 1
fi
