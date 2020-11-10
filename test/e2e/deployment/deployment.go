package deployment

import (
	"context"
	"time"

	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"

	testenv "github.com/nmstate/kubernetes-nmstate/test/env"
)

func GetEventually(deploymentKey types.NamespacedName) AsyncAssertion {
	return Eventually(func() (appsv1.Deployment, error) {
		deployment := appsv1.Deployment{}
		err := testenv.Client.Get(context.TODO(), deploymentKey, &deployment)
		return deployment, err
	}, 180*time.Second, 1*time.Second)
}

// GetDeployment returns a deployment matching passing in deployment Name and Namespace
func Get(deploymentKey types.NamespacedName) (appsv1.Deployment, error) {
	var deployment appsv1.Deployment
	err := testenv.Client.Get(context.TODO(), deploymentKey, &deployment)
	return deployment, err
}
