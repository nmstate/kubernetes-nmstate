/*
Copyright The Kubernetes NMState Authors.


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package names

import (
	"os"
)

// ManifestDir is the directory where manifests are located.
var ManifestDir = "./bindata"

// NMStateResourceName is the name of the CR that the operator will reconcile
const NMStateResourceName = "nmstate"

// Relationship labels
const ComponentLabelKey = "app.kubernetes.io/component"
const PartOfLabelKey = "app.kubernetes.io/part-of"
const VersionLabelKey = "app.kubernetes.io/version"
const ManagedByLabelKey = "app.kubernetes.io/managed-by"

func IncludeRelationshipLabels(labels map[string]string) map[string]string {
	if labels == nil {
		labels = map[string]string{}
	}

	mapLabelKeys := map[string]string{
		"COMPONENT":  ComponentLabelKey,
		"PART_OF":    PartOfLabelKey,
		"VERSION":    VersionLabelKey,
		"MANAGED_BY": ManagedByLabelKey,
	}

	for key, label := range mapLabelKeys {
		envVar := os.Getenv(key)
		if envVar != "" {
			labels[label] = envVar
		}
	}

	return labels
}
