package daemonset

import (
	"context"
	"time"

	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	testenv "github.com/nmstate/kubernetes-nmstate/test/env"
)

func GetEventually(daemonSetKey types.NamespacedName) AsyncAssertion {
	return EventuallyWithOffset(1, func() (appsv1.DaemonSet, error) {
		daemonSet := appsv1.DaemonSet{}
		err := testenv.Client.Get(context.TODO(), daemonSetKey, &daemonSet)
		return daemonSet, err
	}, 180*time.Second, 1*time.Second)
}

func GetEventuallyError(daemonSetKey types.NamespacedName) AsyncAssertion {
	return EventuallyWithOffset(1, func() error {
		daemonSet := appsv1.DaemonSet{}
		err := testenv.Client.Get(context.TODO(), daemonSetKey, &daemonSet)
		return err
	}, 180*time.Second, 1*time.Second)
}

func GetConsistently(daemonSetKey types.NamespacedName) AsyncAssertion {
	return ConsistentlyWithOffset(1, func() (appsv1.DaemonSet, error) {
		daemonSet := appsv1.DaemonSet{}
		err := testenv.Client.Get(context.TODO(), daemonSetKey, &daemonSet)
		return daemonSet, err
	}, 15*time.Second, 1*time.Second)
}

func GetConsistentlyError(daemonSetKey types.NamespacedName) AsyncAssertion {
	return ConsistentlyWithOffset(1, func() (appsv1.DaemonSet, error) {
		daemonSet := appsv1.DaemonSet{}
		err := testenv.Client.Get(context.TODO(), daemonSetKey, &daemonSet)
		return daemonSet, err
	}, 15*time.Second, 1*time.Second)
}

// GetDaemonSetList returns a DaemonSetList matching the labels passed
func GetList(filteringLabels map[string]string) (appsv1.DaemonSetList, error) {
	ds := appsv1.DaemonSetList{}
	err := testenv.Client.List(context.TODO(), &ds, &client.ListOptions{LabelSelector: labels.SelectorFromSet(filteringLabels)})
	return ds, err
}

// GetDaemonSet returns a DaemonSet matching the passed in DaemonSet name and namespace
func Get(daemonSetKey types.NamespacedName) (appsv1.DaemonSet, error) {
	var daemonSet appsv1.DaemonSet
	err := testenv.Client.Get(context.TODO(), daemonSetKey, &daemonSet)
	return daemonSet, err
}
