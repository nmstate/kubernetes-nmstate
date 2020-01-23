#!/bin/bash -e
version_file=version/version.go
# If we don't pass a version just show current one
if [ -z "$1" ]; then
    grep = $version_file | sed -r 's/.*= \"(.*)"$$/v\1/g'
# else change it
else
    sed -i "s/= \".*\"$/= \"$1\"/g" $version_file
fi
