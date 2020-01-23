#!/bin/bash -e
currentver="$(go version | sed 's/.*go\(.*\) .*/\1/')"
minimumver="$(cat go.mod | sed -n 3p | sed 's/go \(.*\)/\1/')"
if [ "$(printf '%s\n' "$minimumver" "$currentver" | sort -V | head -n1)" != "$minimumver" ]; then
    echo "This project requires minimum $minimumver go version, your go versin is $currentver"
    exit 1
fi
