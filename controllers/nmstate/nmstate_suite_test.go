package nmstate

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/onsi/ginkgo/reporters"
)

func copyFile(src, dst string) error {
	var err error
	var srcfd *os.File
	var dstfd *os.File
	var srcinfo os.FileInfo

	// create dst directory if needed
	if _, err := os.Stat(dst); os.IsNotExist(err) {
		if err := os.MkdirAll(dst, os.ModePerm); err != nil {
			return err
		}
	}
	_, fileName := filepath.Split(src)
	dst = dst + fileName
	if srcfd, err = os.Open(src); err != nil {
		return err
	}
	defer srcfd.Close()

	if dstfd, err = os.Create(dst); err != nil {
		return err
	}
	defer dstfd.Close()

	if _, err = io.Copy(dstfd, srcfd); err != nil {
		return err
	}
	if srcinfo, err = os.Stat(src); err != nil {
		return err
	}
	return os.Chmod(dst, srcinfo.Mode())
}

func copyManifests() error {
	srcToDest := map[string]string{
		"../../deploy/crds/nmstate.io_nodenetworkconfigurationenactments.yaml": "./testdata/kubernetes-nmstate/crds/",
		"../../deploy/crds/nmstate.io_nodenetworkconfigurationpolicies.yaml":   "./testdata/kubernetes-nmstate/crds/",
		"../../deploy/crds/nmstate.io_nodenetworkstates.yaml":                  "./testdata/kubernetes-nmstate/crds/",
		"../../deploy/handler/namespace.yaml":                                  "./testdata/kubernetes-nmstate/namespace/",
		"../../deploy/handler/operator.yaml":                                   "./testdata/kubernetes-nmstate/handler/handler.yaml",
		"../../deploy/handler/service_account.yaml":                            "./testdata/kubernetes-nmstate/rbac/",
		"../../deploy/handler/role.yaml":                                       "./testdata/kubernetes-nmstate/rbac/",
		"../../deploy/handler/role_binding.yaml":                               "./testdata/kubernetes-nmstate/rbac/",
	}

	for src, dest := range srcToDest {
		if err := copyFile(src, dest); err != nil {
			return err
		}
	}
	return nil
}

var _ = BeforeSuite(func() {
	err := copyManifests()
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	err := os.RemoveAll("./testdata/kubernetes-nmstate")
	Expect(err).ToNot(HaveOccurred())
})

func TestUnit(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit.controller-nmstate-nmstate_suite_test.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "NMState Controller Test Suite", []Reporter{junitReporter})
}
