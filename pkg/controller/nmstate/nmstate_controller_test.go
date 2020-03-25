package nmstate

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

var _ = Describe("NMstate controller reconcile", func() {
	var (
		cl                  client.Client
		reconciler          ReconcileNMstate
		existingNMstateName = "nmstate01"
		nmState             = nmstatev1alpha1.NMstate{
			ObjectMeta: metav1.ObjectMeta{
				Name: existingNMstateName,
			},
		}
	)
	BeforeEach(func() {
		s := scheme.Scheme
		s.AddKnownTypes(nmstatev1alpha1.SchemeGroupVersion,
			&nmstatev1alpha1.NMstate{},
		)

		objs := []runtime.Object{&nmState}

		// Create a fake client to mock API calls.
		cl = fake.NewFakeClientWithScheme(s, objs...)

		reconciler.client = cl
	})
	Context("when nmstate is not found", func() {
		var (
			request reconcile.Request
		)
		BeforeEach(func() {
			request.Name = "not-present-nmstate"
		})
		It("should return empty result", func() {
			result, err := reconciler.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(reconcile.Result{}))
		})
	})
	Context("when nmstate is found", func() {
		var (
			request reconcile.Request
		)
		It("should return a Result", func() {
			result, err := reconciler.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(reconcile.Result{}))
		})
	})
})
