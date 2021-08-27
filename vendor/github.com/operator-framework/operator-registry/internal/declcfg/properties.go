package declcfg

import (
	"fmt"

	"github.com/operator-framework/operator-registry/internal/property"
)

func parseProperties(props []property.Property) (*property.Properties, error) {
	out, err := property.Parse(props)
	if err != nil {
		return nil, err
	}

	channels := map[string]struct{}{}
	for _, ch := range out.Channels {
		if _, ok := channels[ch.Name]; ok {
			return nil, propertyDuplicateError{typ: property.TypeChannel, key: ch.Name}
		}
		channels[ch.Name] = struct{}{}
	}
	return out, nil
}

type propertyDuplicateError struct {
	typ string
	key string
}

func (e propertyDuplicateError) Error() string {
	return fmt.Sprintf("duplicate property of type %q found with key %q", e.typ, e.key)
}
