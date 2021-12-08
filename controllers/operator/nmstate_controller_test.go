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

	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/nmstate/kubernetes-nmstate/api/names"
	nmstatev1 "github.com/nmstate/kubernetes-nmstate/api/v1"
)

var _ = Describe("NMState controller reconcile", func() {
	var (
		cl                  client.Client
		reconciler          NMStateReconciler
		existingNMStateName = "nmstate"
		handlerNodeSelector = map[string]string{"selector_1": "value_1", "selector_2": "value_2"}
		handlerTolerations  = []corev1.Toleration{
			{
				Effect:   "NoSchedule",
				Key:      "node.kubernetes.io/special-toleration",
				Operator: "Exists",
			},
		}
		infraNodeSelector = map[string]string{"webhookselector_1": "value_1", "webhookselector_2": "value_2"}
		infraTolerations  = []corev1.Toleration{
			{
				Effect:   "NoSchedule",
				Key:      "node.kubernetes.io/special-webhook-toleration",
				Operator: "Exists",
			},
		}
		nmstate = nmstatev1.NMState{
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
		s.AddKnownTypes(nmstatev1.GroupVersion,
			&nmstatev1.NMState{},
			&nmstatev1.NMStateList{},
		)
		objs := []runtime.Object{&nmstate}
		// Create a fake client to mock API calls.
		cl = fake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(objs...).Build()
		names.ManifestDir = manifestsDir
		reconciler.Client = cl
		reconciler.Scheme = s
		reconciler.Log = ctrl.Log.WithName("controllers").WithName("NMState")
		os.Setenv("HANDLER_NAMESPACE", handlerNamespace)
		os.Setenv("RELATED_IMAGE_HANDLER_IMAGE", handlerImage)
		os.Setenv("HANDLER_IMAGE_PULL_POLICY", imagePullPolicy)
		os.Setenv("HANDLER_PREFIX", handlerPrefix)
	})
	AfterEach(func() {
		err := os.RemoveAll(manifestsDir)
		Expect(err).ToNot(HaveOccurred())
	})

	Context("when additional CR is created", func() {
		var (
			request ctrl.Request
		)
		BeforeEach(func() {
			request.Name = "nmstate-two"
		})
		It("should return empty result", func() {
			result, err := reconciler.Reconcile(context.Background(), request)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))
		})
		It("and should delete the second one", func() {
			_, err := reconciler.Reconcile(context.Background(), request)
			Expect(err).ToNot(HaveOccurred())
			nmstateList := &nmstatev1.NMStateList{}
			err = cl.List(context.TODO(), nmstateList, &client.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(nmstateList.Items)).To(Equal(1))
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
			result, err := reconciler.Reconcile(context.Background(), request)
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
			_, err := reconciler.Reconcile(context.Background(), request)
			Expect(err).To(HaveOccurred())
		})
	})
	Context("when operator spec has a NodeSelector", func() {
		var (
			request ctrl.Request
		)
		BeforeEach(func() {
			s := scheme.Scheme
			s.AddKnownTypes(nmstatev1.GroupVersion,
				&nmstatev1.NMState{},
			)
			// set NodeSelector field in operator Spec
			nmstate.Spec.NodeSelector = handlerNodeSelector
			objs := []runtime.Object{&nmstate}
			// Create a fake client to mock API calls.
			cl = fake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(objs...).Build()
			reconciler.Client = cl
			request.Name = existingNMStateName
			result, err := reconciler.Reconcile(context.Background(), request)
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
	Context("when operator spec has Tolerations", func() {
		var (
			request ctrl.Request
		)
		BeforeEach(func() {
			s := scheme.Scheme
			s.AddKnownTypes(nmstatev1.GroupVersion,
				&nmstatev1.NMState{},
			)
			// set Tolerations field in operator Spec
			nmstate.Spec.Tolerations = handlerTolerations
			objs := []runtime.Object{&nmstate}
			// Create a fake client to mock API calls.
			cl = fake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(objs...).Build()
			reconciler.Client = cl
			request.Name = existingNMStateName
			result, err := reconciler.Reconcile(context.Background(), request)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))
		})
		It("should add Tolerations to handler daemonset", func() {
			ds := &appsv1.DaemonSet{}
			handlerKey := types.NamespacedName{Namespace: handlerNamespace, Name: handlerPrefix + "-nmstate-handler"}
			err := cl.Get(context.TODO(), handlerKey, ds)
			Expect(err).ToNot(HaveOccurred())
			Expect(allTolerationsPresent(handlerTolerations, ds.Spec.Template.Spec.Tolerations)).To(BeTrue())
		})
		It("should NOT add Tolerations to webhook deployment", func() {
			deployment := &appsv1.Deployment{}
			webhookKey := types.NamespacedName{Namespace: handlerNamespace, Name: handlerPrefix + "-nmstate-webhook"}
			err := cl.Get(context.TODO(), webhookKey, deployment)
			Expect(err).ToNot(HaveOccurred())
			Expect(anyTolerationsPresent(handlerTolerations, deployment.Spec.Template.Spec.Tolerations)).To(BeFalse())
		})
	})
	Context("when operator spec has a InfraNodeSelector", func() {
		var (
			request ctrl.Request
		)
		BeforeEach(func() {
			s := scheme.Scheme
			s.AddKnownTypes(nmstatev1.GroupVersion,
				&nmstatev1.NMState{},
			)
			// set InfraNodeSelector field in operator Spec
			nmstate.Spec.InfraNodeSelector = infraNodeSelector
			objs := []runtime.Object{&nmstate}
			// Create a fake client to mock API calls.
			cl = fake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(objs...).Build()
			reconciler.Client = cl
			request.Name = existingNMStateName
			result, err := reconciler.Reconcile(context.Background(), request)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))
		})
		It("should add InfraNodeSelector to webhook deployment", func() {
			deployment := &appsv1.Deployment{}
			webhookKey := types.NamespacedName{Namespace: handlerNamespace, Name: handlerPrefix + "-nmstate-webhook"}
			err := cl.Get(context.TODO(), webhookKey, deployment)
			Expect(err).ToNot(HaveOccurred())
			for k, v := range infraNodeSelector {
				Expect(deployment.Spec.Template.Spec.NodeSelector).To(HaveKeyWithValue(k, v))
			}
		})
		It("should add InfraNodeSelector to certmanager deployment", func() {
			deployment := &appsv1.Deployment{}
			certManagerKey := types.NamespacedName{Namespace: handlerNamespace, Name: handlerPrefix + "-nmstate-cert-manager"}
			err := cl.Get(context.TODO(), certManagerKey, deployment)
			Expect(err).ToNot(HaveOccurred())
			for k, v := range infraNodeSelector {
				Expect(deployment.Spec.Template.Spec.NodeSelector).To(HaveKeyWithValue(k, v))
			}
		})
		It("should NOT add InfraNodeSelector to handler daemonset", func() {
			ds := &appsv1.DaemonSet{}
			handlerKey := types.NamespacedName{Namespace: handlerNamespace, Name: handlerPrefix + "-nmstate-handler"}
			err := cl.Get(context.TODO(), handlerKey, ds)
			Expect(err).ToNot(HaveOccurred())
			for k, v := range infraNodeSelector {
				Expect(ds.Spec.Template.Spec.NodeSelector).ToNot(HaveKeyWithValue(k, v))
			}
		})
	})
	Context("when operator spec has InfraTolerations", func() {
		var (
			request ctrl.Request
		)
		BeforeEach(func() {
			s := scheme.Scheme
			s.AddKnownTypes(nmstatev1.GroupVersion,
				&nmstatev1.NMState{},
			)
			// set Tolerations field in operator Spec
			nmstate.Spec.InfraTolerations = infraTolerations
			objs := []runtime.Object{&nmstate}
			// Create a fake client to mock API calls.
			cl = fake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(objs...).Build()
			reconciler.Client = cl
			request.Name = existingNMStateName
			result, err := reconciler.Reconcile(context.Background(), request)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))
		})
		It("should add InfraTolerations to webhook deployment", func() {
			deployment := &appsv1.Deployment{}
			webhookKey := types.NamespacedName{Namespace: handlerNamespace, Name: handlerPrefix + "-nmstate-webhook"}
			err := cl.Get(context.TODO(), webhookKey, deployment)
			Expect(err).ToNot(HaveOccurred())
			Expect(allTolerationsPresent(infraTolerations, deployment.Spec.Template.Spec.Tolerations)).To(BeTrue())
		})
		It("should add InfraTolerations to cert-manager deployment", func() {
			deployment := &appsv1.Deployment{}
			certManagerKey := types.NamespacedName{Namespace: handlerNamespace, Name: handlerPrefix + "-nmstate-cert-manager"}
			err := cl.Get(context.TODO(), certManagerKey, deployment)
			Expect(err).ToNot(HaveOccurred())
			Expect(allTolerationsPresent(infraTolerations, deployment.Spec.Template.Spec.Tolerations)).To(BeTrue())
		})
		It("should NOT add InfraTolerations to handler daemonset", func() {
			ds := &appsv1.DaemonSet{}
			handlerKey := types.NamespacedName{Namespace: handlerNamespace, Name: handlerPrefix + "-nmstate-handler"}
			err := cl.Get(context.TODO(), handlerKey, ds)
			Expect(err).ToNot(HaveOccurred())
			Expect(anyTolerationsPresent(infraTolerations, ds.Spec.Template.Spec.Tolerations)).To(BeFalse())
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
		"../../deploy/crds/nmstate.io_nodenetworkconfigurationenactments.yaml": "kubernetes-nmstate/crds/",
		"../../deploy/crds/nmstate.io_nodenetworkconfigurationpolicies.yaml":   "kubernetes-nmstate/crds/",
		"../../deploy/crds/nmstate.io_nodenetworkstates.yaml":                  "kubernetes-nmstate/crds/",
		"../../deploy/handler/namespace.yaml":                                  "kubernetes-nmstate/namespace/",
		"../../deploy/handler/operator.yaml":                                   "kubernetes-nmstate/handler/handler.yaml",
		"../../deploy/handler/service_account.yaml":                            "kubernetes-nmstate/rbac/",
		"../../deploy/handler/role.yaml":                                       "kubernetes-nmstate/rbac/",
		"../../deploy/handler/role_binding.yaml":                               "kubernetes-nmstate/rbac/",
	}

	for src, dest := range srcToDest {
		if err := copyManifest(src, filepath.Join(manifestsDir, dest)); err != nil {
			return err
		}
	}
	return nil
}

func checkTolerationInList(toleration corev1.Toleration, tolerationList []corev1.Toleration) bool {
	found := false
	for _, currentToleration := range tolerationList {
		if isSuperset(toleration, currentToleration) {
			found = true
			break
		}
	}
	return found
}

// isSuperset checks whether ss tolerates a superset of t.
func isSuperset(ss, t corev1.Toleration) bool {
	if apiequality.Semantic.DeepEqual(&t, &ss) {
		return true
	}

	if !isKeyMatching(t, ss) {
		return false
	}

	if !isEffectMatching(t, ss) {
		return false
	}

	if ss.Effect == corev1.TaintEffectNoExecute {
		if ss.TolerationSeconds != nil {
			if t.TolerationSeconds == nil ||
				*t.TolerationSeconds > *ss.TolerationSeconds {
				return false
			}
		}
	}

	switch ss.Operator {
	case corev1.TolerationOpEqual, "": // empty operator means Equal
		return t.Operator == corev1.TolerationOpEqual && t.Value == ss.Value
	case corev1.TolerationOpExists:
		return true
	default:
		return false
	}
}

//allTolerationsPresent check if all tolerations from toBeCheckedTolerations are superseded by actualTolerations.
func allTolerationsPresent(toBeCheckedTolerations []corev1.Toleration, actualTolerations []corev1.Toleration) bool {
	tolerationsFound := true
	for _, toleration := range toBeCheckedTolerations {
		tolerationsFound = tolerationsFound && checkTolerationInList(toleration, actualTolerations)
	}
	return tolerationsFound
}

//anyTolerationsPresent check whether any tolerations from toBeCheckedTolerations are part of actualTolerations.
func anyTolerationsPresent(toBeCheckedTolerations []corev1.Toleration, actualTolerations []corev1.Toleration) bool {
	tolerationsFound := false
	for _, toleration := range toBeCheckedTolerations {
		tolerationsFound = tolerationsFound || checkTolerationInList(toleration, actualTolerations)
	}
	return tolerationsFound
}

// isKeyMatching check if tolerations arguments match the toleration keys.
func isKeyMatching(a, b corev1.Toleration) bool {
	if a.Key == b.Key || (b.Key == "" && b.Operator == corev1.TolerationOpExists) {
		return true
	}
	return false
}

// isEffectMatching check if tolerations arguments match the effects
func isEffectMatching(a, b corev1.Toleration) bool {
	// An empty effect means match all effects.
	return a.Effect == b.Effect || b.Effect == ""
}
