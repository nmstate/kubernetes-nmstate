package node

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/onsi/ginkgo/reporters"
)

func TestUnit(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit.controller-node-node_suite_test.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Node Controller Test Suite", []Reporter{junitReporter})
}
