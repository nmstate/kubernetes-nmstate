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

func GetIntVarWithDefault(name string, defaultValue int) int {
	value := os.Getenv(name)
	intValue, err := strconv.Atoi(value)
	if err != nil {
		intValue = defaultValue
	}
	return intValue
}
