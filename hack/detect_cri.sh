#!/usr/bin/env bash

set -e

determine_cri_bin() {
    if podman ps >/dev/null 2>&1; then
        echo podman
    elif docker ps >/dev/null 2>&1; then
        echo docker
    else
        echo ""
    fi
}

determine_cri_bin
