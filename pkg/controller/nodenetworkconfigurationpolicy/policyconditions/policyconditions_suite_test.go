package policyconditions

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/onsi/ginkgo/reporters"
)

func TestUnit(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit.controller-nodenetworkconfigurationpolicy-policyconditions-policyconditions_suite_test.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Conditions Test Suite", []Reporter{junitReporter})
}
