package selectors

import (
	"github.com/go-logr/logr"

	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
)

type Selectors struct {
	client client.Client
	policy nmstatev1beta1.NodeNetworkConfigurationPolicy
	logger logr.Logger
}

func NewFromPolicy(client client.Client, policy nmstatev1beta1.NodeNetworkConfigurationPolicy) Selectors {
	selectors := Selectors{
		client: client,
		policy: policy,
	}
	selectors.logger = logf.Log.WithName("policy/selectors").WithValues("policy", policy.Name)
	return selectors
}
