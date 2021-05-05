package enactment

import (
	"context"

	nmstateapi "github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
	enactmentconditions "github.com/nmstate/kubernetes-nmstate/pkg/enactmentstatus/conditions"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CountByPolicy(cli client.Client, policy *nmstatev1beta1.NodeNetworkConfigurationPolicy) (int, enactmentconditions.ConditionCount, error) {
	enactments := nmstatev1beta1.NodeNetworkConfigurationEnactmentList{}
	policyLabelFilter := client.MatchingLabels{nmstateapi.EnactmentPolicyLabel: policy.GetName()}
	err := cli.List(context.TODO(), &enactments, policyLabelFilter)
	if err != nil {
		return 0, nil, errors.Wrap(err, "getting enactment list failed")
	}
	enactmentCount := enactmentconditions.Count(enactments, policy.Generation)
	return len(enactments.Items), enactmentCount, nil
}
