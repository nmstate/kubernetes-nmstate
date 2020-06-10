package daemonset

import (
	"context"
	"time"

	. "github.com/onsi/gomega"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetEventually(daemonSetKey types.NamespacedName) AsyncAssertion {
	return Eventually(func() (appsv1.DaemonSet, error) {
		daemonSet := appsv1.DaemonSet{}
		err := framework.Global.Client.Get(context.TODO(), daemonSetKey, &daemonSet)
		return daemonSet, err
	}, 180*time.Second, 1*time.Second)
}

// GetDaemonSetList returns a DaemonSetList matching the labels passed
func GetList(filteringLabels map[string]string) (appsv1.DaemonSetList, error) {
	ds := appsv1.DaemonSetList{}
	err := framework.Global.Client.List(context.TODO(), &ds, &client.ListOptions{LabelSelector: labels.SelectorFromSet(filteringLabels)})
	return ds, err
}

// GetDaemonSet returns a DaemonSet matching the passed in DaemonSet name and namespace
func Get(daemonSetKey types.NamespacedName) (appsv1.DaemonSet, error) {
	var daemonSet appsv1.DaemonSet
	err := framework.Global.Client.Get(context.TODO(), daemonSetKey, &daemonSet)
	return daemonSet, err
}
