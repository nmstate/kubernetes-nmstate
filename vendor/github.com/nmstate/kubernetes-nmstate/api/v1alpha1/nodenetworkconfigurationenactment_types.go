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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	nmstateapiv2 "github.com/nmstate/nmstate/rust/src/go/api/v2"

	"github.com/nmstate/kubernetes-nmstate/api/names"
	"github.com/nmstate/kubernetes-nmstate/api/shared"
)

// +kubebuilder:object:root=true

// NodeNetworkConfigurationEnactmentList contains a list of NodeNetworkConfigurationEnactment
type NodeNetworkConfigurationEnactmentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NodeNetworkConfigurationEnactment `json:"items"`
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=nodenetworkconfigurationenactments,shortName=nnce,scope=Cluster
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[?(@.status==\"True\")].type",description="Status"
//nolint:lll
// +kubebuilder:printcolumn:name="Status Age",type="date",JSONPath=".status.conditions[?(@.status==\"True\")].lastTransitionTime",description="Status Age"
// +kubebuilder:deprecatedversion

// NodeNetworkConfigurationEnactment is the Schema for the nodenetworkconfigurationenactments API
type NodeNetworkConfigurationEnactment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Status shared.NodeNetworkConfigurationEnactmentStatus `json:"status,omitempty"`
}

func NewEnactment(nodeName string, policy *NodeNetworkConfigurationPolicy) NodeNetworkConfigurationEnactment {
	enactment := NodeNetworkConfigurationEnactment{
		ObjectMeta: metav1.ObjectMeta{
			Name: shared.EnactmentKey(nodeName, policy.Name).Name,
			OwnerReferences: []metav1.OwnerReference{
				{Name: policy.Name, Kind: policy.TypeMeta.Kind, APIVersion: policy.TypeMeta.APIVersion, UID: policy.UID},
			},
			// Associate policy with the enactment using labels
			Labels: names.IncludeRelationshipLabels(map[string]string{shared.EnactmentPolicyLabel: policy.Name}),
		},
		Status: shared.NodeNetworkConfigurationEnactmentStatus{
			DesiredState: nmstateapiv2.NetworkState{},
			Conditions:   shared.ConditionList{},
		},
	}

	for _, conditionType := range shared.NodeNetworkConfigurationEnactmentConditionTypes {
		enactment.Status.Conditions.Set(conditionType, corev1.ConditionUnknown, "", "")
	}
	return enactment
}

func init() {
	SchemeBuilder.Register(&NodeNetworkConfigurationEnactment{}, &NodeNetworkConfigurationEnactmentList{})
}
