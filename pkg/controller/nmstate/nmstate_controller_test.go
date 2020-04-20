package nmstate

import (
	"context"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1beta1"
	"github.com/nmstate/kubernetes-nmstate/pkg/names"
)

var _ = Describe("NMState controller reconcile", func() {
	var (
		cl                  client.Client
		reconciler          ReconcileNMState
		existingNMStateName = "nmstate"
		dsNodeSelector      = map[string]string{"selector_1": "value_1", "selector_2": "value_2"}
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
	)
	BeforeEach(func() {
		s := scheme.Scheme
		s.AddKnownTypes(nmstatev1beta1.SchemeGroupVersion,
			&nmstatev1beta1.NMState{},
		)
		objs := []runtime.Object{&nmstate}
		// Create a fake client to mock API calls.
		cl = fake.NewFakeClientWithScheme(s, objs...)
		names.ManifestDir = "./testdata"
		reconciler.client = cl
		reconciler.scheme = scheme.Scheme
		os.Setenv("HANDLER_NAMESPACE", handlerNamespace)
		os.Setenv("HANDLER_IMAGE", handlerImage)
		os.Setenv("HANDLER_IMAGE_PULL_POLICY", imagePullPolicy)
		os.Setenv("HANDLER_PREFIX", handlerPrefix)

	})
	Context("when CR is wrong name", func() {
		var (
			request reconcile.Request
		)
		BeforeEach(func() {
			request.Name = "not-present-node"
		})
		It("should return empty result", func() {
			result, err := reconciler.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(reconcile.Result{}))
		})
	})
	Context("when an nmstate is found", func() {
		var (
			request reconcile.Request
		)
		BeforeEach(func() {
			request.Name = existingNMStateName
		})
		It("should return a Result", func() {
			result, err := reconciler.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(reconcile.Result{}))
		})
	})
	Context("when one of manifest directory is empty", func() {
		var (
			request reconcile.Request
		)
		BeforeEach(func() {
			request.Name = existingNMStateName
		})

		AfterEach(func() {
			copyManifests()
		})
		It("should return error", func() {
			os.RemoveAll("./testdata/kubernetes-nmstate/crds/")
			os.MkdirAll("./testdata/kubernetes-nmstate/crds/", os.ModePerm)
			_, err := reconciler.Reconcile(request)
			Expect(err).To(HaveOccurred())
		})
	})
	Context("with NodeSelector defined in Operator spec", func() {
		var (
			request reconcile.Request
		)
		BeforeEach(func() {
			s := scheme.Scheme
			s.AddKnownTypes(nmstatev1beta1.SchemeGroupVersion,
				&nmstatev1beta1.NMState{},
			)
			// set NodeSelector field in operator Spec
			nmstate.Spec.NodeSelector = dsNodeSelector
			objs := []runtime.Object{&nmstate}
			// Create a fake client to mock API calls.
			cl = fake.NewFakeClientWithScheme(s, objs...)
			reconciler.client = cl
			request.Name = existingNMStateName
		})
		It("should add NodeSelector labels to Daemonsets spec", func() {
			result, err := reconciler.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(reconcile.Result{}))
			// verify NodeSelector value for both worker and master DS
			ds := &appsv1.DaemonSet{}
			for _, c := range []string{"worker", "master"} {
				cl.Get(context.TODO(), types.NamespacedName{Namespace: handlerNamespace, Name: handlerPrefix + "-nmstate-handler-" + c}, ds)
				for k, v := range dsNodeSelector {
					Expect(ds.Spec.Template.Spec.NodeSelector).To(HaveKeyWithValue(k, v))
				}
			}
		})
	})

})
