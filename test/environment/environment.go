package environment

import (
	"os"
	"strconv"
)

func GetVarWithDefault(name string, defaultValue string) string {
	value := os.Getenv(name)
	if len(value) == 0 {
		value = defaultValue
	}
	return value
}

func GetBoolVarWithDefault(name string, defaultValue bool) bool {
	value := os.Getenv(name)
	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		boolValue = defaultValue
	}
	return boolValue
}
