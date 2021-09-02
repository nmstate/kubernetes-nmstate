package names

import (
	"os"
)

// ManifestDir is the directory where manifests are located.
var ManifestDir = "./bindata"

// NMStateResourceName is the name of the CR that the operator will reconcile
const NMStateResourceName = "nmstate"

// Relationship labels
const COMPONENT_LABEL_KEY = "app.kubernetes.io/component"
const PART_OF_LABEL_KEY = "app.kubernetes.io/part-of"
const VERSION_LABEL_KEY = "app.kubernetes.io/version"
const MANAGED_BY_LABEL_KEY = "app.kubernetes.io/managed-by"

func IncludeRelationshipLabels(labels map[string]string) map[string]string {
	if labels == nil {
		labels = map[string]string{}
	}

	mapLabelKeys := map[string]string{
		"COMPONENT":  COMPONENT_LABEL_KEY,
		"PART_OF":    PART_OF_LABEL_KEY,
		"VERSION":    VERSION_LABEL_KEY,
		"MANAGED_BY": MANAGED_BY_LABEL_KEY,
	}

	for key, label := range mapLabelKeys {
		envVar := os.Getenv(key)
		if envVar != "" {
			labels[label] = envVar
		}
	}

	return labels
}
