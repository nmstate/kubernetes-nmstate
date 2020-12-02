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
	valueStr := os.Getenv(name)
	value, err := strconv.ParseInt(valueStr, 10, 64)
	if err != nil {
		return defaultValue
	}
	return int(value)
}
