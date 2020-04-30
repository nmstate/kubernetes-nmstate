package nmstate

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/onsi/ginkgo/reporters"
)

func TestUnit(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit.controller-nmstate-nmstate_suite_test.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "NMstate Controller Test Suite", []Reporter{junitReporter})
}
