// +build !ignore_autogenerated

// This file was autogenerated by openapi-gen. Do not edit it manually!

package v1alpha1

import (
	spec "github.com/go-openapi/spec"
	common "k8s.io/kube-openapi/pkg/common"
)

func GetOpenAPIDefinitions(ref common.ReferenceCallback) map[string]common.OpenAPIDefinition {
	return map[string]common.OpenAPIDefinition{
		"./pkg/apis/nmstate/v1alpha1.Condition":                            schema_pkg_apis_nmstate_v1alpha1_Condition(ref),
		"./pkg/apis/nmstate/v1alpha1.NodeNetworkConfigurationPolicy":       schema_pkg_apis_nmstate_v1alpha1_NodeNetworkConfigurationPolicy(ref),
		"./pkg/apis/nmstate/v1alpha1.NodeNetworkConfigurationPolicySpec":   schema_pkg_apis_nmstate_v1alpha1_NodeNetworkConfigurationPolicySpec(ref),
		"./pkg/apis/nmstate/v1alpha1.NodeNetworkConfigurationPolicyStatus": schema_pkg_apis_nmstate_v1alpha1_NodeNetworkConfigurationPolicyStatus(ref),
		"./pkg/apis/nmstate/v1alpha1.NodeNetworkState":                     schema_pkg_apis_nmstate_v1alpha1_NodeNetworkState(ref),
		"./pkg/apis/nmstate/v1alpha1.NodeNetworkStateStatus":               schema_pkg_apis_nmstate_v1alpha1_NodeNetworkStateStatus(ref),
		"./pkg/apis/nmstate/v1alpha1.State":                                schema_pkg_apis_nmstate_v1alpha1_State(ref),
	}
}

func schema_pkg_apis_nmstate_v1alpha1_Condition(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Type: []string{"object"},
				Properties: map[string]spec.Schema{
					"type": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"status": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"reason": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"message": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"lastHearbeatTime": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("k8s.io/apimachinery/pkg/apis/meta/v1.Time"),
						},
					},
					"lastTransitionTime": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("k8s.io/apimachinery/pkg/apis/meta/v1.Time"),
						},
					},
				},
				Required: []string{"type", "status"},
			},
		},
		Dependencies: []string{
			"k8s.io/apimachinery/pkg/apis/meta/v1.Time"},
	}
}

func schema_pkg_apis_nmstate_v1alpha1_NodeNetworkConfigurationPolicy(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "NodeNetworkConfigurationPolicy is the Schema for the nodenetworkconfigurationpolicies API",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"kind": {
						SchemaProps: spec.SchemaProps{
							Description: "Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"apiVersion": {
						SchemaProps: spec.SchemaProps{
							Description: "APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#resources",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"metadata": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"),
						},
					},
					"spec": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("./pkg/apis/nmstate/v1alpha1.NodeNetworkConfigurationPolicySpec"),
						},
					},
					"status": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("./pkg/apis/nmstate/v1alpha1.NodeNetworkConfigurationPolicyStatus"),
						},
					},
				},
			},
		},
		Dependencies: []string{
			"./pkg/apis/nmstate/v1alpha1.NodeNetworkConfigurationPolicySpec", "./pkg/apis/nmstate/v1alpha1.NodeNetworkConfigurationPolicyStatus", "k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"},
	}
}

func schema_pkg_apis_nmstate_v1alpha1_NodeNetworkConfigurationPolicySpec(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "NodeNetworkConfigurationPolicySpec defines the desired state of NodeNetworkConfigurationPolicy",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"nodeSelector": {
						SchemaProps: spec.SchemaProps{
							Description: "NodeSelector is a selector which must be true for the policy to be applied to the node. Selector which must match a node's labels for the policy to be scheduled on that node. More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/",
							Type:        []string{"object"},
							AdditionalProperties: &spec.SchemaOrBool{
								Allows: true,
								Schema: &spec.Schema{
									SchemaProps: spec.SchemaProps{
										Type:   []string{"string"},
										Format: "",
									},
								},
							},
						},
					},
					"desiredState": {
						SchemaProps: spec.SchemaProps{
							Description: "The desired configuration of the policy",
							Ref:         ref("./pkg/apis/nmstate/v1alpha1.State"),
						},
					},
				},
			},
		},
		Dependencies: []string{
			"./pkg/apis/nmstate/v1alpha1.State"},
	}
}

func schema_pkg_apis_nmstate_v1alpha1_NodeNetworkConfigurationPolicyStatus(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "NodeNetworkConfigurationPolicyStatus defines the observed state of NodeNetworkConfigurationPolicy",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"conditions": {
						SchemaProps: spec.SchemaProps{
							Type: []string{"array"},
							Items: &spec.SchemaOrArray{
								Schema: &spec.Schema{
									SchemaProps: spec.SchemaProps{
										Ref: ref("./pkg/apis/nmstate/v1alpha1.Condition"),
									},
								},
							},
						},
					},
				},
			},
		},
		Dependencies: []string{
			"./pkg/apis/nmstate/v1alpha1.Condition"},
	}
}

func schema_pkg_apis_nmstate_v1alpha1_NodeNetworkState(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "NodeNetworkState is the Schema for the nodenetworkstates API",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"kind": {
						SchemaProps: spec.SchemaProps{
							Description: "Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"apiVersion": {
						SchemaProps: spec.SchemaProps{
							Description: "APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#resources",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"metadata": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"),
						},
					},
					"status": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("./pkg/apis/nmstate/v1alpha1.NodeNetworkStateStatus"),
						},
					},
				},
			},
		},
		Dependencies: []string{
			"./pkg/apis/nmstate/v1alpha1.NodeNetworkStateStatus", "k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"},
	}
}

func schema_pkg_apis_nmstate_v1alpha1_NodeNetworkStateStatus(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "NodeNetworkStateStatus is the status of the NodeNetworkState of a specific node",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"currentState": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("./pkg/apis/nmstate/v1alpha1.State"),
						},
					},
					"conditions": {
						SchemaProps: spec.SchemaProps{
							Type: []string{"array"},
							Items: &spec.SchemaOrArray{
								Schema: &spec.Schema{
									SchemaProps: spec.SchemaProps{
										Ref: ref("./pkg/apis/nmstate/v1alpha1.Condition"),
									},
								},
							},
						},
					},
					"enactments": {
						SchemaProps: spec.SchemaProps{
							Type: []string{"array"},
							Items: &spec.SchemaOrArray{
								Schema: &spec.Schema{
									SchemaProps: spec.SchemaProps{
										Ref: ref("./pkg/apis/nmstate/v1alpha1.Enactment"),
									},
								},
							},
						},
					},
				},
			},
		},
		Dependencies: []string{
			"./pkg/apis/nmstate/v1alpha1.Condition", "./pkg/apis/nmstate/v1alpha1.Enactment", "./pkg/apis/nmstate/v1alpha1.State"},
	}
}

func schema_pkg_apis_nmstate_v1alpha1_State(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "State contains the namestatectl yaml [1] as string instead of golang struct so we don't need to be in sync with the schema.\n\n[1] https://github.com/nmstate/nmstate/blob/master/libnmstate/schemas/operational-state.yaml",
				Type:        []string{"object"},
			},
		},
	}
}
