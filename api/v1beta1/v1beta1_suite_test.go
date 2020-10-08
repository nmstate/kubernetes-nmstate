package v1beta1

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/onsi/ginkgo/reporters"
)

func TestUnit(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit.apis-nmstate-v1beta1-v1beta1_suite_test.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "API Test Suite", []Reporter{junitReporter})
}
