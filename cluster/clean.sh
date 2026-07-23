#!/bin/bash

set -ex

source ./cluster/lima.sh
lima::ensure_linux
source ./cluster/sync-operator.sh

function eventually {
    timeout=15
    interval=5
    cmd=$@
    echo "Checking eventually $cmd"
    while ! $cmd; do
        if [ $timeout -le 0 ]; then
            return 1
        fi
        sleep $interval
        timeout=$(( $timeout - $interval ))
    done
}

function clean() {
    echo 'Cleaning up ...'

    clean_operator
}

# Use labels so we don't care about prefixes
function isRemoved {
    output=$($kubectl get $1 -n $2 -l $3 2>&1)
    [[ ! $output =~ ".*No resources found.*" ]]
}

function isHandlerRemoved {
    isRemoved daemonset ${HANDLER_NAMESPACE:-nmstate} app=kubernetes-nmstate
}

function isWebhookRemoved {
    isRemoved deployment ${HANDLER_NAMESPACE:-nmstate} app=kubernetes-nmstate
}

function wait_removed() {
    if ! eventually isHandlerRemoved; then
        echo "Handler hasn't been removed within the given timeout"
        exit 1
    fi

    if ! eventually isWebhookRemoved; then
        echo "Webhook hasn't been removed within the given timeout"
        exit 1
    fi

}

clean
wait_removed
