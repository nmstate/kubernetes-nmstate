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

package enactment

import (
	"context"

	nmstateapi "github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1 "github.com/nmstate/kubernetes-nmstate/api/v1"
	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
	enactmentconditions "github.com/nmstate/kubernetes-nmstate/pkg/enactmentstatus/conditions"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CountByPolicy(cli client.Reader, policy *nmstatev1.NodeNetworkConfigurationPolicy) (int, enactmentconditions.ConditionCount, error) {
	enactments := nmstatev1beta1.NodeNetworkConfigurationEnactmentList{}
	policyLabelFilter := client.MatchingLabels{nmstateapi.EnactmentPolicyLabel: policy.GetName()}
	err := cli.List(context.TODO(), &enactments, policyLabelFilter)
	if err != nil {
		return 0, nil, errors.Wrap(err, "getting enactment list failed")
	}
	enactmentCount := enactmentconditions.Count(enactments, policy.Generation)
	return len(enactments.Items), enactmentCount, nil
}
