package conditions

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/onsi/ginkgo/reporters"
)

func TestUnit(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit.controller-nodenetworkconfigurationpolicy-enactmentstatus-conditions_suite_test.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Enactment Status Conditions Test Suite", []Reporter{junitReporter})
}
