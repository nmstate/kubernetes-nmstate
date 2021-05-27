package enactmentstatus

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/onsi/ginkgo/reporters"
)

func TestUnit(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit.enactmentstatus-enactmentstatus_suite_test.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Enactment Status Test Suite", []Reporter{junitReporter})
}
