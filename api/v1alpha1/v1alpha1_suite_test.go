package v1alpha1

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/onsi/ginkgo/reporters"
)

func TestUnit(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit.apis-nmstate-v1alpha1-v1alpha1_suite_test.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "API Test Suite", []Reporter{junitReporter})
}
