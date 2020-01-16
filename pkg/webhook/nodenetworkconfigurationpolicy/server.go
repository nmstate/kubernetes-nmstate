package nodenetworkconfigurationpolicy

import (
	"sigs.k8s.io/controller-runtime/pkg/manager"

	webhookserver "github.com/nmstate/kubernetes-nmstate/pkg/webhook/server"
)

const (
	webhookName = "nmstate"
)

func Add(mgr manager.Manager) error {
	server := webhookserver.New(mgr, webhookName,
		webhookserver.WithHook("/nodenetworkconfigurationpolicies-mutate", resetConditionsHook()))
	return add(mgr, server)
}

// add adds a new Webhook to mgr with r as the webhook.Server
func add(mgr manager.Manager, s manager.Runnable) error {
	mgr.Add(s)
	return nil
}
