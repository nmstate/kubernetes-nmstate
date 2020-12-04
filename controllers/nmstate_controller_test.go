package controllers

import (
	"context"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
	"github.com/nmstate/kubernetes-nmstate/pkg/names"
)

var _ = Describe("NMState controller reconcile", func() {
	var (
		cl                  client.Client
		reconciler          NMStateReconciler
		existingNMStateName = "nmstate"
		handlerNodeSelector = map[string]string{"selector_1": "value_1", "selector_2": "value_2"}
		nmstate             = nmstatev1beta1.NMState{
			ObjectMeta: metav1.ObjectMeta{
				Name: existingNMStateName,
				UID:  "12345",
			},
		}
		handlerPrefix    = "handler"
		handlerNamespace = "nmstate"
		handlerImage     = "quay.io/some_image"
		imagePullPolicy  = "Always"
		manifestsDir     = ""
	)
	BeforeEach(func() {
		var err error
		manifestsDir, err = ioutil.TempDir("/tmp", "knmstate-test-manifests")
		Expect(err).ToNot(HaveOccurred())
		err = copyManifests(manifestsDir)
		Expect(err).ToNot(HaveOccurred())

		s := scheme.Scheme
		s.AddKnownTypes(nmstatev1beta1.GroupVersion,
			&nmstatev1beta1.NMState{},
		)
		objs := []runtime.Object{&nmstate}
		// Create a fake client to mock API calls.
		cl = fake.NewFakeClientWithScheme(s, objs...)
		names.ManifestDir = manifestsDir
		reconciler.Client = cl
		reconciler.Scheme = s
		reconciler.Log = ctrl.Log.WithName("controllers").WithName("NMState")
		os.Setenv("HANDLER_NAMESPACE", handlerNamespace)
		os.Setenv("HANDLER_IMAGE", handlerImage)
		os.Setenv("HANDLER_IMAGE_PULL_POLICY", imagePullPolicy)
		os.Setenv("HANDLER_PREFIX", handlerPrefix)
	})
	AfterEach(func() {
		err := os.RemoveAll(manifestsDir)
		Expect(err).ToNot(HaveOccurred())
	})

	Context("when CR is wrong name", func() {
		var (
			request ctrl.Request
		)
		BeforeEach(func() {
			request.Name = "not-present-node"
		})
		It("should return empty result", func() {
			result, err := reconciler.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))
		})
	})
	Context("when an nmstate is found", func() {
		var (
			request ctrl.Request
		)
		BeforeEach(func() {
			request.Name = existingNMStateName
		})
		It("should return a Result", func() {
			result, err := reconciler.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))
		})
	})
	Context("when one of manifest directory is empty", func() {
		var (
			request ctrl.Request
		)
		BeforeEach(func() {
			request.Name = existingNMStateName
			crdsPath := filepath.Join(manifestsDir, "kubernetes-nmstate/crds/")
			dir, err := ioutil.ReadDir(crdsPath)
			Expect(err).ToNot(HaveOccurred())
			for _, d := range dir {
				os.RemoveAll(filepath.Join(crdsPath, d.Name()))
			}
		})
		It("should return error", func() {
			_, err := reconciler.Reconcile(request)
			Expect(err).To(HaveOccurred())
		})
	})
	Context("when operator spec has a NodeSelector", func() {
		var (
			request ctrl.Request
		)
		BeforeEach(func() {
			s := scheme.Scheme
			s.AddKnownTypes(nmstatev1beta1.GroupVersion,
				&nmstatev1beta1.NMState{},
			)
			// set NodeSelector field in operator Spec
			nmstate.Spec.NodeSelector = handlerNodeSelector
			objs := []runtime.Object{&nmstate}
			// Create a fake client to mock API calls.
			cl = fake.NewFakeClientWithScheme(s, objs...)
			reconciler.Client = cl
			request.Name = existingNMStateName
			result, err := reconciler.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))
		})
		It("should add NodeSelector to handler daemonset", func() {
			ds := &appsv1.DaemonSet{}
			handlerKey := types.NamespacedName{Namespace: handlerNamespace, Name: handlerPrefix + "-nmstate-handler"}
			err := cl.Get(context.TODO(), handlerKey, ds)
			Expect(err).ToNot(HaveOccurred())
			for k, v := range handlerNodeSelector {
				Expect(ds.Spec.Template.Spec.NodeSelector).To(HaveKeyWithValue(k, v))
			}
		})
		It("should NOT add NodeSelector to webhook deployment", func() {
			deployment := &appsv1.Deployment{}
			webhookKey := types.NamespacedName{Namespace: handlerNamespace, Name: handlerPrefix + "-nmstate-webhook"}
			err := cl.Get(context.TODO(), webhookKey, deployment)
			Expect(err).ToNot(HaveOccurred())
			for k, v := range handlerNodeSelector {
				Expect(deployment.Spec.Template.Spec.NodeSelector).ToNot(HaveKeyWithValue(k, v))
			}
		})
	})

})

func copyManifest(src, dst string) error {
	var err error
	var srcfd *os.File
	var dstfd *os.File
	var srcinfo os.FileInfo

	dstDir := dst
	fileName := ""
	dstIsAManifest := strings.HasSuffix(dst, ".yaml")
	if dstIsAManifest {
		dstDir, fileName = filepath.Split(dstDir)
	}

	// create dst directory if needed
	if _, err := os.Stat(dstDir); os.IsNotExist(err) {
		if err := os.MkdirAll(dstDir, os.ModePerm); err != nil {
			return err
		}
	}
	if fileName == "" {
		_, fileName = filepath.Split(src)
	}
	dst = filepath.Join(dstDir, fileName)
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

func copyManifests(manifestsDir string) error {
	srcToDest := map[string]string{
		"../deploy/crds/nmstate.io_nodenetworkconfigurationenactments.yaml": "kubernetes-nmstate/crds/",
		"../deploy/crds/nmstate.io_nodenetworkconfigurationpolicies.yaml":   "kubernetes-nmstate/crds/",
		"../deploy/crds/nmstate.io_nodenetworkstates.yaml":                  "kubernetes-nmstate/crds/",
		"../deploy/handler/namespace.yaml":                                  "kubernetes-nmstate/namespace/",
		"../deploy/handler/operator.yaml":                                   "kubernetes-nmstate/handler/handler.yaml",
		"../deploy/handler/service_account.yaml":                            "kubernetes-nmstate/rbac/",
		"../deploy/handler/role.yaml":                                       "kubernetes-nmstate/rbac/",
		"../deploy/handler/role_binding.yaml":                               "kubernetes-nmstate/rbac/",
	}

	for src, dest := range srcToDest {
		if err := copyManifest(src, filepath.Join(manifestsDir, dest)); err != nil {
			return err
		}
	}
	return nil
}
