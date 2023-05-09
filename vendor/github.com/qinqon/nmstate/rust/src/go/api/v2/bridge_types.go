package nmstate

// +k8s:deepcopy-gen=true
type LinuxBridgeInterface struct {
	// +kubebuilder:validation:Type=object
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:pruning:PreserveUnknownFields
	Bridge    *LinuxBridgeConfig `json:"bridge,omitempty",json:"ovs-bridge,omitempty"`
	OvsBridge *LinuxBridgeConfig `json:"linux-bridge,omitempty",json:"ovs-bridge,omitempty"`
}

// +k8s:deepcopy-gen=true
type OvsBridgeInterface struct {
	// +kubebuilder:validation:Type=object
	// +kubebuilder:validation:Schemaless
	Bridge    *OvsBridgeConfig `json:"bridge,omitempty",json:"ovs-bridge,omitempty"`
	OvsBridge *OvsBridgeConfig `json:"ovs-bridge,omitempty",json:"ovs-bridge,omitempty"`
}
