#!/bin/bash -xe

${KUBECTL} apply -f ${MANIFESTS_DIR}/namespace.yaml

mkdir -p test_logs/e2e/operator
unset GOFLAGS && ${OPERATOR_SDK} test local ./test/e2e/operator \
	--kubeconfig ${KUBECONFIG} \
	--namespace ${OPERATOR_NAMESPACE} \
	--up-local \
	--go-test-flags "${TEST_ARGS}"
