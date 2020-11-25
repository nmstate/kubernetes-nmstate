package appregistry

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
)

type registrySpecifier struct {
}

func (p *registrySpecifier) Parse(specifiers []string) ([]*Source, error) {
	sources := make([]*Source, 0)
	allErrors := []error{}

	for _, specifier := range specifiers {
		source, err := p.ParseOne(specifier)
		if err != nil {
			allErrors = append(allErrors, err)
			continue
		}

		sources = append(sources, source)
	}

	err := utilerrors.NewAggregate(allErrors)
	return sources, err
}

// ParseOne constructs a Source objects from a given pipe delimited
// representation of an operator source. The format is specified as shown below.

// {base url with cnr prefix}|{quay registry namespace}|{secret namespace/secret name}
//
// Secret is optional.
func (*registrySpecifier) ParseOne(specifier string) (*Source, error) {
	splits := strings.Split(specifier, "|")

	// If leading or trailing delimiter is specified, then handle it by removing
	// the empty string from the slice.
	values := make([]string, 0)
	for _, value := range splits {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}

		values = append(values, value)
	}

	if len(values) < 2 || len(values) > 3 {
		return nil, fmt.Errorf("The source specified is invalid - %s", specifier)
	}

	var secret types.NamespacedName
	// If secret has been specified let's handle it.
	if len(values) == 3 {
		s := values[2]
		split := strings.Split(s, "/")
		if len(split) != 2 {
			return nil, fmt.Errorf("invalid source, secret specified is malformed - %s", s)
		}

		secret.Namespace = strings.TrimSpace(split[0])
		secret.Name = strings.TrimSpace(split[1])
	}

	source := &Source{
		Endpoint:          values[0],
		RegistryNamespace: values[1],
		Secret:            secret,
	}

	return source, nil
}
