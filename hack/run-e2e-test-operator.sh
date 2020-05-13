#!/bin/bash -xe

mkdir -p test_logs/e2e/operator
unset GOFLAGS && ${OPERATOR_SDK} test local ./test/e2e/operator \
	--kubeconfig ${KUBECONFIG} \
	--operator-namespace ${OPERATOR_NAMESPACE} \
	--no-setup \
	--go-test-flags "${TEST_ARGS}"
