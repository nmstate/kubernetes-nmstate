#!/bin/bash -xe

suite=$1

if [ -z "${suite}" ]; then
    exit 1
fi

mkdir -p test_logs/e2e/${suite}
unset GOFLAGS && ${OPERATOR_SDK} test local ./test/e2e/${suite} \
	--kubeconfig ${KUBECONFIG} \
	--namespace ${HANDLER_NAMESPACE} \
	--no-setup \
	--go-test-flags "${TEST_ARGS}"
