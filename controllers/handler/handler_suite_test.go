package controllers

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/onsi/ginkgo/reporters"
)

func TestUnit(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit.controllers_handler_suite_test.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Controllers Handler Test Suite", []Reporter{junitReporter})
}
