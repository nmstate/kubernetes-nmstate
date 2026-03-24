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

package metrics

import (
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/nmstate/kubernetes-nmstate/api/shared"
)

func activeConditionType(conditions shared.ConditionList) shared.ConditionType {
	for _, c := range conditions {
		if c.Status == corev1.ConditionTrue {
			return c.Type
		}
	}
	return ""
}

// conditionChangePredicate returns a predicate that triggers on create, delete,
// and on updates only when the active condition type changes.
// getConditions extracts the ConditionList from the concrete object type.
func conditionChangePredicate(getConditions func(client.Object) (shared.ConditionList, bool)) predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(event.CreateEvent) bool {
			return true
		},
		DeleteFunc: func(event.DeleteEvent) bool {
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldConditions, ok := getConditions(e.ObjectOld)
			if !ok {
				return true
			}
			newConditions, ok := getConditions(e.ObjectNew)
			if !ok {
				return true
			}
			return activeConditionType(oldConditions) != activeConditionType(newConditions)
		},
		GenericFunc: func(event.GenericEvent) bool {
			return false
		},
	}
}
