package selectors

import (
	"github.com/go-logr/logr"

	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

type Request struct {
	client   client.Client
	policy   nmstatev1alpha1.NodeNetworkConfigurationPolicy
	nodeName string
	logger   logr.Logger
}

func New(client client.Client, nodeName string, policy nmstatev1alpha1.NodeNetworkConfigurationPolicy) Request {
	request := Request{
		client:   client,
		policy:   policy,
		nodeName: nodeName,
	}
	request.logger = logf.Log.WithName("policy/selectors").WithValues("enactment", nmstatev1alpha1.EnactmentKey(nodeName, policy.Name).Name)
	return request
}
