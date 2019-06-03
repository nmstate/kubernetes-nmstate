package v1

import (
	yaml "github.com/ghodss/yaml"
)

// Serialize t as yaml into output
func (t State) MarshalJSON() (output []byte, err error) {
	return yaml.YAMLToJSON([]byte(t))
}

// Stores directly the json string into t
func (t *State) UnmarshalJSON(b []byte) error {
	output, err := yaml.JSONToYAML(b)
	if err != nil {
		return err
	}
	*t = State(output)
	return nil
}

func (t State) String() string {
	return string(t)
}
