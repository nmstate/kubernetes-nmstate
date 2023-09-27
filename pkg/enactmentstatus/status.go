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

package enactmentstatus

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
)

var (
	log = logf.Log.WithName("enactmentstatus")
)

func Update(cli client.Client, key types.NamespacedName, statusSetter func(*nmstate.NodeNetworkConfigurationEnactmentStatus)) error {
	logger := log.WithValues("enactment", key.Name)
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		instance := &nmstatev1beta1.NodeNetworkConfigurationEnactment{}
		err := cli.Get(context.TODO(), key, instance)
		if err != nil {
			return errors.Wrap(err, "getting enactment failed")
		}

		statusSetter(&instance.Status)

		logger.Info(fmt.Sprintf("status: %+v", instance.Status))

		return cli.Status().Update(context.TODO(), instance)
	})
}

func IsProgressing(conditions *nmstate.ConditionList) bool {
	progressingCondition := conditions.Find(nmstate.NodeNetworkConfigurationEnactmentConditionProgressing)
	if progressingCondition != nil && progressingCondition.Status == corev1.ConditionTrue {
		return true
	}
	return false
}
