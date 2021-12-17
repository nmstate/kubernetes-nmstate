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
