package appregistry

import (
	"fmt"

	"k8s.io/apimachinery/pkg/types"
)

type Source struct {
	// Endpoint points to the base URL of remote AppRegistry server.
	Endpoint string

	// RegistryNamespace is the namespace in remote registry where the operator
	// metadata is located.
	RegistryNamespace string

	// Secret is the kubernetes secret object that contains the authorization
	// token that can be used to access private repositories.
	Secret types.NamespacedName
}

func (s *Source) String() string {
	return fmt.Sprintf("%s/%s", s.Endpoint, s.RegistryNamespace)
}

func (s *Source) IsSecretSpecified() bool {
	if s.Secret.Name == "" || s.Secret.Namespace == "" {
		return false
	}

	return true
}
