package nodenetworkconfigurationpolicy

import (
	"github.com/go-logr/logr"
	"github.com/pkg/errors"

	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	certificate "github.com/nmstate/kubernetes-nmstate/pkg/webhook/certificate"
)

const (
	webhookName = "nmstate"
)

type server struct {
	mgr           manager.Manager
	webhookServer *webhook.Server
	log           logr.Logger
}

// Add creates a new Conditions Mutating Webhook and adds it to the Manager. The Manager will set fields on the Webhook
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newServer(mgr))
}

func newServer(mgr manager.Manager) *server {
	s := &server{
		webhookServer: &webhook.Server{
			Port:    8443,
			CertDir: "/etc/webhook/certs/",
		},
		mgr: mgr,
		log: logf.Log.WithName("webhook/nodenetworkconfigurationpolicies/server"),
	}
	s.webhookServer.Register("/nodenetworkconfigurationpolicies-mutate", resetConditionsHook())
	return s
}

// add adds a new Webhook to mgr with r as the webhook.Server
func add(mgr manager.Manager, s *server) error {
	mgr.Add(s)
	return nil
}

func (s *server) Start(stop <-chan struct{}) error {
	s.log.Info("Starting nodenetworkconfigurationpolicy webhook server")

	// We have only one webhook so we just take the first one
	certManager, err := certificate.NewManager(s.mgr, webhookName, s.webhookServer.CertDir, "tls.crt", "tls.key")
	if err != nil {
		return errors.Wrap(err, "failed creating new webhook cert manager")
	}

	err = certManager.Start()
	if err != nil {
		return errors.Wrap(err, "failed starting webhook cert manager")
	}
	defer certManager.Stop()

	err = s.webhookServer.Start(stop)
	if err != nil {
		return errors.Wrap(err, "failed starting webhook server")
	}
	return nil
}

func (s *server) InjectFunc(f inject.Func) error {
	return s.webhookServer.InjectFunc(f)
}

func (s *server) NeedLeaderElection() bool {
	return s.webhookServer.NeedLeaderElection()
}
