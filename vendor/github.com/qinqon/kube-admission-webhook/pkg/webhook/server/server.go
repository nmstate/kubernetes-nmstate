package server

import (
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/qinqon/kube-admission-webhook/pkg/certificate"
	"github.com/qinqon/kube-admission-webhook/pkg/certificate/triple"
)

type Server struct {
	webhookServer *webhook.Server
	certManager   *certificate.Manager
	log           logr.Logger
}

type ServerModifier func(w *webhook.Server)

// Add creates a new Conditions Mutating Webhook and adds it to the Manager. The Manager will set fields on the Webhook
// and Start it when the Manager is Started.
func New(client client.Client, webhookName string, webhookType certificate.WebhookType, caRotateInterval time.Duration, serverOpts ...ServerModifier) *Server {
	s := &Server{
		webhookServer: &webhook.Server{
			Port:    8443,
			CertDir: "/etc/webhook/certs/",
		},
		certManager: certificate.NewManager(client, webhookName, webhookType, caRotateInterval),
		log:         logf.Log.WithName("webhook/server"),
	}
	s.UpdateOpts(serverOpts...)
	s.webhookServer.Register("/readyz", healthz.CheckHandler{Checker: healthz.Ping})
	return s
}

func WithHook(path string, hook *webhook.Admission) ServerModifier {
	return func(s *webhook.Server) {
		s.Register(path, hook)
	}
}

func WithPort(port int) ServerModifier {
	return func(s *webhook.Server) {
		s.Port = port
	}
}

func WithCertDir(certDir string) ServerModifier {
	return func(s *webhook.Server) {
		s.CertDir = certDir
	}
}

//updates Server parameters using ServerModifier functions. Once the manager is started these parameters cannot be updated
func (s *Server) UpdateOpts(serverOpts ...ServerModifier) {
	for _, serverOpt := range serverOpts {
		serverOpt(s.webhookServer)
	}
}

func (s *Server) Add(mgr manager.Manager) error {
	err := s.certManager.Add(mgr)
	if err != nil {
		return errors.Wrap(err, "failed adding certificate manager to controller-runtime manager")
	}
	err = mgr.Add(s)
	if err != nil {
		return errors.Wrap(err, "failed adding webhook server to controller-runtime manager")
	}
	return nil
}

func (s *Server) checkTLS() error {

	keyPath := path.Join(s.webhookServer.CertDir, corev1.TLSPrivateKeyKey)
	_, err := os.Stat(keyPath)
	if err != nil {
		return errors.Wrap(err, "failed checking TLS key file stats")
	}

	certsPath := path.Join(s.webhookServer.CertDir, corev1.TLSCertKey)
	_, err = os.Stat(certsPath)
	if err != nil {
		return errors.Wrap(err, "failed checking TLS cert file stats")
	}

	key, err := ioutil.ReadFile(path.Join(s.webhookServer.CertDir, corev1.TLSPrivateKeyKey))
	if err != nil {
		return errors.Wrap(err, "failed reading for TLS key")
	}

	certPEM, err := ioutil.ReadFile(path.Join(s.webhookServer.CertDir, corev1.TLSCertKey))
	if err != nil {
		return errors.Wrap(err, "failed reading for TLS cert")
	}

	caPEM, err := s.certManager.CABundle()
	if err != nil {
		return errors.Wrap(err, "failed to retrieve CA cert")
	}

	err = triple.VerifyTLS(certPEM, key, caPEM)
	if err != nil {
		return errors.Wrapf(err, "failed verifying %s/%s", certsPath, keyPath)
	}

	return nil
}

func (s *Server) waitForTLSReadiness() error {
	return wait.PollImmediate(5*time.Second, 5*time.Minute, func() (bool, error) {
		err := s.checkTLS()
		if err != nil {
			utilruntime.HandleError(err)
			return false, nil
		}
		return true, nil
	})
}

func (s *Server) Start(stop <-chan struct{}) error {
	s.log.Info("Starting nodenetworkconfigurationpolicy webhook server")

	err := s.waitForTLSReadiness()
	if err != nil {
		return errors.Wrap(err, "failed watting for ready TLS key/cert")
	}

	err = s.webhookServer.Start(stop)
	if err != nil {
		return errors.Wrap(err, "failed starting webhook server")
	}
	return nil
}

func (s *Server) InjectFunc(f inject.Func) error {
	return s.webhookServer.InjectFunc(f)
}

func (s *Server) NeedLeaderElection() bool {
	return s.webhookServer.NeedLeaderElection()
}
