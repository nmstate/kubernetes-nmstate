/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit.controller_suite_test.xml")
	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}, junitReporter})
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.LoggerTo(GinkgoWriter, true))
	err := copyManifests()
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	err := os.RemoveAll("./testdata/kubernetes-nmstate")
	Expect(err).ToNot(HaveOccurred())
})

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
		"../deploy/crds/nmstate.io_nodenetworkconfigurationenactments_crd.yaml": "./testdata/kubernetes-nmstate/crds/",
		"../deploy/crds/nmstate.io_nodenetworkconfigurationpolicies_crd.yaml":   "./testdata/kubernetes-nmstate/crds/",
		"../deploy/crds/nmstate.io_nodenetworkstates_crd.yaml":                  "./testdata/kubernetes-nmstate/crds/",
		"../deploy/handler/namespace.yaml":                                      "./testdata/kubernetes-nmstate/namespace/",
		"../deploy/handler/operator.yaml":                                       "./testdata/kubernetes-nmstate/handler/handler.yaml",
		"../deploy/handler/service_account.yaml":                                "./testdata/kubernetes-nmstate/rbac/",
		"../deploy/handler/role.yaml":                                           "./testdata/kubernetes-nmstate/rbac/",
		"../deploy/handler/role_binding.yaml":                                   "./testdata/kubernetes-nmstate/rbac/",
	}

	for src, dest := range srcToDest {
		if err := copyFile(src, dest); err != nil {
			return err
		}
	}
	return nil
}
