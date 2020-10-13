package nodenetworkconfigurationpolicy

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/onsi/ginkgo/reporters"
)

func TestUnit(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit.controller-nodenetworkconfigurationpolicy-nodenetworkconfigurationpolicy_suite_test.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "NodeNetworkConfigurationPolicy controller Test Suite", []Reporter{junitReporter})
}
