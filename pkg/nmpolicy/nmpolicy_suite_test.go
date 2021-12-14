package nmpolicy

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/onsi/ginkgo/reporters"
)

func TestUnit(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit.nmpolicy_suite_test.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "NMPolicy Test Suite", []Reporter{junitReporter})
}
