#!/bin/bash -xe

main() {
    TARGET="$0"
    TARGET="${TARGET#./}"
    TARGET="${TARGET%.*}"
    TARGET="${TARGET#*.}"
    echo "TARGET=$TARGET"
    export TARGET

    echo "Setup Go paths"
    cd ..
    export GOROOT=/usr/local/go
    export GOPATH=$(pwd)/go
    export PATH=$GOPATH/bin:$GOROOT/bin:$PATH
    mkdir -p $GOPATH

    echo "Install Go 1.11"
    export GIMME_GO_VERSION=1.11
    mkdir -p /gimme
    curl -sL https://raw.githubusercontent.com/travis-ci/gimme/master/gimme | HOME=/gimme bash >> /etc/profile.d/gimme.sh
    source /etc/profile.d/gimme.sh

    echo "Install operator repository to the right place"
    mkdir -p $GOPATH/src/github.com/nmstate
    mkdir -p $GOPATH/pkg
    # symlink does not work with make we need a copy
    cp -rf $(pwd)/kubernetes-nmstate $GOPATH/src/github.com/nmstate/
    cd $GOPATH/src/github.com/nmstate/kubernetes-nmstate

    echo "Install operator-sdk"
    curl -JL https://github.com/operator-framework/operator-sdk/releases/download/v0.8.0/operator-sdk-v0.8.0-x86_64-linux-gnu -o /usr/bin/operator-sdk
    chmod +x /usr/bin/operator-sdk

    echo "Run functional tests"
    exec automation/test.sh
}

[[ "${BASH_SOURCE[0]}" == "$0" ]] && main "$@"
