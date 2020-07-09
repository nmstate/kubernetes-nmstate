#!/bin/bash -xe

mkdir -p test_logs/e2e/handler
unset GOFLAGS && ${OPERATOR_SDK} test local ./test/e2e/handler \
	--kubeconfig ${KUBECONFIG} \
	--namespace ${HANDLER_NAMESPACE} \
	--no-setup \
	--go-test-flags "${TEST_ARGS}  --test-suite-params=\"$POLARION_TEST_SUITE_PARAMS\""
