package selectors

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/onsi/ginkgo/reporters"
)

func TestUnit(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit.controller-nodenetworkconfigurationpolicy-selectors-selectors_suite_test.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Policy Selectors Test Suite", []Reporter{junitReporter})
}
