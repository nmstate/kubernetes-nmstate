/*
Copyright The Kubernetes NMState Authors.


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

package tls

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

// callbackRecorder captures invocations of SecurityProfileWatcher.OnProfileChange.
type callbackRecorder struct {
	calls []callbackCall
}

type callbackCall struct {
	oldProfile   TLSProfileSpec
	newProfile   TLSProfileSpec
	oldAdherence TLSAdherencePolicy
	newAdherence TLSAdherencePolicy
}

func (r *callbackRecorder) fn() func(ctx context.Context, oldProfile, newProfile TLSProfileSpec, oldAdherence, newAdherence TLSAdherencePolicy) {
	return func(_ context.Context, oldProfile, newProfile TLSProfileSpec, oldAdherence, newAdherence TLSAdherencePolicy) {
		r.calls = append(r.calls, callbackCall{
			oldProfile:   oldProfile,
			newProfile:   newProfile,
			oldAdherence: oldAdherence,
			newAdherence: newAdherence,
		})
	}
}

func reconcileRequest() ctrl.Request {
	return ctrl.Request{NamespacedName: types.NamespacedName{Name: apiServerName}}
}

var _ = Describe("SecurityProfileWatcher.Reconcile", func() {
	ctx := context.Background()

	It("returns no error and does not call the callback when the APIServer is absent", func() {
		rec := &callbackRecorder{}
		w := &SecurityProfileWatcher{
			Client:          newFakeClient(),
			OnProfileChange: rec.fn(),
		}
		res, err := w.Reconcile(ctx, reconcileRequest())
		Expect(err).NotTo(HaveOccurred())
		Expect(res).To(Equal(ctrl.Result{}))
		Expect(rec.calls).To(BeEmpty())
	})

	It("fires the callback on the first reconcile when initial state differs from current", func() {
		obj := newAPIServerUnstructured()
		setTLSSecurityProfile(obj, map[string]interface{}{"type": "Modern"})
		setTLSAdherence(obj, "StrictAllComponents")

		rec := &callbackRecorder{}
		w := &SecurityProfileWatcher{
			Client:                    newFakeClient(obj),
			InitialTLSProfileSpec:     *TLSProfiles[TLSProfileIntermediateType],
			InitialTLSAdherencePolicy: TLSAdherenceLegacyAdheringComponentsOnly,
			OnProfileChange:           rec.fn(),
		}

		_, err := w.Reconcile(ctx, reconcileRequest())
		Expect(err).NotTo(HaveOccurred())
		Expect(rec.calls).To(HaveLen(1))
		Expect(rec.calls[0].oldProfile).To(Equal(*TLSProfiles[TLSProfileIntermediateType]))
		Expect(rec.calls[0].newProfile).To(Equal(*TLSProfiles[TLSProfileModernType]))
		Expect(rec.calls[0].oldAdherence).To(Equal(TLSAdherenceLegacyAdheringComponentsOnly))
		Expect(rec.calls[0].newAdherence).To(Equal(TLSAdherenceStrictAllComponents))

		// And the watcher updates its internal state so that a second
		// reconcile against the same cluster state is a no-op.
		_, err = w.Reconcile(ctx, reconcileRequest())
		Expect(err).NotTo(HaveOccurred())
		Expect(rec.calls).To(HaveLen(1))
	})

	It("does not fire the callback when current state already matches initial state", func() {
		obj := newAPIServerUnstructured()
		setTLSSecurityProfile(obj, map[string]interface{}{"type": "Intermediate"})

		rec := &callbackRecorder{}
		w := &SecurityProfileWatcher{
			Client:                    newFakeClient(obj),
			InitialTLSProfileSpec:     *TLSProfiles[TLSProfileIntermediateType],
			InitialTLSAdherencePolicy: TLSAdherencePolicy(""),
			OnProfileChange:           rec.fn(),
		}
		_, err := w.Reconcile(ctx, reconcileRequest())
		Expect(err).NotTo(HaveOccurred())
		Expect(rec.calls).To(BeEmpty())
	})

	It("fires the callback when only the profile changes", func() {
		obj := newAPIServerUnstructured()
		setTLSSecurityProfile(obj, map[string]interface{}{"type": "Modern"})
		setTLSAdherence(obj, "StrictAllComponents")

		rec := &callbackRecorder{}
		w := &SecurityProfileWatcher{
			Client:                    newFakeClient(obj),
			InitialTLSProfileSpec:     *TLSProfiles[TLSProfileIntermediateType],
			InitialTLSAdherencePolicy: TLSAdherenceStrictAllComponents,
			OnProfileChange:           rec.fn(),
		}
		_, err := w.Reconcile(ctx, reconcileRequest())
		Expect(err).NotTo(HaveOccurred())
		Expect(rec.calls).To(HaveLen(1))
		Expect(rec.calls[0].oldAdherence).To(Equal(rec.calls[0].newAdherence))
		Expect(rec.calls[0].oldProfile).NotTo(Equal(rec.calls[0].newProfile))
	})

	It("fires the callback when only the adherence changes", func() {
		obj := newAPIServerUnstructured()
		setTLSSecurityProfile(obj, map[string]interface{}{"type": "Intermediate"})
		setTLSAdherence(obj, "StrictAllComponents")

		rec := &callbackRecorder{}
		w := &SecurityProfileWatcher{
			Client:                    newFakeClient(obj),
			InitialTLSProfileSpec:     *TLSProfiles[TLSProfileIntermediateType],
			InitialTLSAdherencePolicy: TLSAdherenceLegacyAdheringComponentsOnly,
			OnProfileChange:           rec.fn(),
		}
		_, err := w.Reconcile(ctx, reconcileRequest())
		Expect(err).NotTo(HaveOccurred())
		Expect(rec.calls).To(HaveLen(1))
		Expect(rec.calls[0].oldProfile).To(Equal(rec.calls[0].newProfile))
		Expect(rec.calls[0].oldAdherence).To(Equal(TLSAdherenceLegacyAdheringComponentsOnly))
		Expect(rec.calls[0].newAdherence).To(Equal(TLSAdherenceStrictAllComponents))
	})

	It("fires the callback when both profile and adherence change", func() {
		obj := newAPIServerUnstructured()
		setTLSSecurityProfile(obj, map[string]interface{}{"type": "Old"})
		setTLSAdherence(obj, "StrictAllComponents")

		rec := &callbackRecorder{}
		w := &SecurityProfileWatcher{
			Client:                    newFakeClient(obj),
			InitialTLSProfileSpec:     *TLSProfiles[TLSProfileIntermediateType],
			InitialTLSAdherencePolicy: TLSAdherenceLegacyAdheringComponentsOnly,
			OnProfileChange:           rec.fn(),
		}
		_, err := w.Reconcile(ctx, reconcileRequest())
		Expect(err).NotTo(HaveOccurred())
		Expect(rec.calls).To(HaveLen(1))
		Expect(rec.calls[0].newProfile).To(Equal(*TLSProfiles[TLSProfileOldType]))
		Expect(rec.calls[0].newAdherence).To(Equal(TLSAdherenceStrictAllComponents))
	})

	It("does not panic when OnProfileChange is nil and still updates internal state", func() {
		obj := newAPIServerUnstructured()
		setTLSSecurityProfile(obj, map[string]interface{}{"type": "Modern"})

		w := &SecurityProfileWatcher{
			Client:                newFakeClient(obj),
			InitialTLSProfileSpec: *TLSProfiles[TLSProfileIntermediateType],
		}
		Expect(func() {
			_, err := w.Reconcile(ctx, reconcileRequest())
			Expect(err).NotTo(HaveOccurred())
		}).NotTo(Panic())
		Expect(w.InitialTLSProfileSpec).To(Equal(*TLSProfiles[TLSProfileModernType]))
	})

	It("returns an error when spec.tlsSecurityProfile cannot be parsed", func() {
		obj := newAPIServerUnstructured()
		obj.Object["spec"] = map[string]interface{}{
			"tlsSecurityProfile": "not-a-map",
		}
		rec := &callbackRecorder{}
		w := &SecurityProfileWatcher{
			Client:          newFakeClient(obj),
			OnProfileChange: rec.fn(),
		}
		_, err := w.Reconcile(ctx, reconcileRequest())
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failed to parse TLS profile"))
		Expect(rec.calls).To(BeEmpty())
	})
})
