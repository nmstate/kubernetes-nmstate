package helper

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const configMapName = "nmstate-config"

// return True if config map name is nmstate-config
func EventIsForNmConfig(meta v1.Object) bool {
	configName := meta.GetName()
	return configName == configMapName
}
