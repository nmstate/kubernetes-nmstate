package helper

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/onsi/ginkgo/reporters"
)

func TestUnit(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit.controller-nodenetworkconfigurationpolicy-helpers-names_suite_test.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Helpers Test Suite", []Reporter{junitReporter})
}
