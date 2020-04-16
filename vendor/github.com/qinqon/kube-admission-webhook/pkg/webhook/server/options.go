package server

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

type ServerModifier func(w *Server)

func WithHook(path string, hook *webhook.Admission) ServerModifier {
	return func(s *Server) {
		s.webhookServer.Register(path, hook)
	}
}

func WithPort(port int) ServerModifier {
	return func(s *Server) {
		s.webhookServer.Port = port
	}
}

func WithCertDir(certDir string) ServerModifier {
	return func(s *Server) {
		s.webhookServer.CertDir = certDir
	}
}

func WithCACert(key types.NamespacedName, field string) ServerModifier {
	return func(s *Server) {
		s.caConfigMapKey = key
		s.caConfigMapField = field
	}
}

func WithK8SCACert() ServerModifier {
	return WithCACert(
		types.NamespacedName{
			Namespace: "kube-system",
			Name:      "extension-apiserver-authentication",
		},
		"client-ca-file",
	)
}

func WithOpenshiftCACert() ServerModifier {
	return WithCACert(
		types.NamespacedName{
			Namespace: "openshift-config",
			Name:      "initial-kube-apiserver-server-ca",
		},
		"ca-bundle.crt",
	)
}

func WithAutoCACert() ServerModifier {
	return func(s *Server) {
		if s.isOpenshift() {
			WithOpenshiftCACert()(s)
		} else {
			WithK8SCACert()(s)
		}
		s.log.Info(fmt.Sprintf("Selected configmap data to generate caBundle {key: '%+v', field: '%s'}", s.caConfigMapKey, s.caConfigMapField))
	}
}

// Return true if it's running at openshift false otherwise, to check it
// it does the programmatic version of `kubectl get co openshift-apiserver`
func (s *Server) isOpenshift() bool {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "config.openshift.io",
		Kind:    "ClusterOperator",
		Version: "v1",
	})
	err := s.mgr.GetClient().Get(context.Background(), client.ObjectKey{
		Name: "openshift-apiserver",
	}, u)
	return err == nil
}
