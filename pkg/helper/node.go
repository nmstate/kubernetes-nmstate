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

package helper

import (
	"github.com/nmstate/kubernetes-nmstate/pkg/environment"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Return true if the event name is the name of
// the pods's node (reading the env var NODE_NAME)
func EventIsForThisNode(meta v1.Object) bool {
	createdNodeName := meta.GetName()
	podNodeName := environment.NodeName()
	// Only reconcile is it's for this pod
	return createdNodeName == podNodeName
}
