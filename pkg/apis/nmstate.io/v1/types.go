package v1

import (
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeNetworkState contains the networking state of the node.
// This would be used as the desired state as well as the current state.
type NodeNetworkState struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              NodeNetworkStateSpec   `json:"spec"`
	Status            NodeNetworkStateStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeNetworkStateList is a list of NodeNetworkState.
type NodeNetworkStateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []NodeNetworkState `json:"items"`
}

// NodeNetworkStateSpec is a description of a NodeNetworkState
// An admin can mark the object as non-managed, signalling the implementation that it
// should compute the desired state of the node,
// but that it should not to attempt to change the operational network state.
type NodeNetworkStateSpec struct {
	Managed bool `json:"managed"`
	// Name of the node reporting this state
	NodeName string `json:"nodeName"`
	// The desired configuration for the node derived from the NodeNetConfPolicy
	DesiredState ConfigurationState `json:"desiredState"`
}

// NodeNetworkStateStatus is the status of the NodeNetworkState of a specific node
type NodeNetworkStateStatus struct {
	Capabilities CapabilityList `json:"capabilities"`
	// Current configuration as well as operational state of the interfaces
	Interfaces InterfaceInfoList `json:"interfaces"`
}

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeNetConfPolicy network configuration policy in the cluster
type NodeNetConfPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              NodeNetConfPolicySpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeNetConfPolicyList is a list of NodeNetConfPolicy
type NodeNetConfPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []NodeNetConfPolicy `json:"items"`
}

// NodeNetConfPolicySpec is a description of a NodeNetConfPolicy
type NodeNetConfPolicySpec struct {
	// Node affinity may be used to select on which nodes the state should be applied
	// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
	Affinity *k8sv1.Affinity `json:"affinity,omitempty"`
	// Node tolerations may be used to select on which nodes the state should NOT be applied
	// More info: https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/
	Tolerations []k8sv1.Toleration `json:"tolerations,omitempty"`
	// List of interfaces on which the configuration should be applied.
	// List must include at least one interface matching rule.
	Match MatchRules `json:"match"`
	// Specification for auto configuration of the matched interfaces.
	// Mutually exclusive with DesiredState.
	AutoConfig *AutoConfigSpec `json:"autoconfig"`
	// Specification of the desired state of the matched interfaces.
	// Mutually exclusive with AutoConfig.
	DesiredState *ConfigurationState `json:"desiredState"`
}

// ConfigurationState holding a list of interfaces and their configuration.
// Used both as desired configuration and actual configuration.
type ConfigurationState struct {
	Interfaces InterfaceSpecList `json:"interfaces"`
}

// InterfaceSpecList list of interfaces and their configuration.
type InterfaceSpecList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []InterfaceSpec `json:"items"`
}

// InterfaceSpec configuration of an interface
type InterfaceSpec struct {
	// Name of the interface
	Name string `json:"name"`
	// Description of the interface
	Description string `json:"description,omitempty"`
	// Type of interface
	Type InterfaceType `json:"type,omitempty"`
	// State of interface
	State InterfaceState `json:"state,omitempty"`
	// MAC address set on the interface
	MACAddress string `json:"macAddress,omitempty"`
	// MTU size in bytes set on the interface
	MTU *uint `json:"mtu,omitempty"`
	// auto-negotiation setting of the interface
	AutoNegotiation *bool `json:"autoNegotiation"`
	// duplex settings of the interface
	Duplex DuplexType `json:"duplex,omitempty"`
	// Link speed in TODO: which units
	LinkSpeed *uint `json:"linkSpeed,omitempty"`
	// Flow control setting of the interface
	FlowControl *bool `json:"flowControl,omitempty"`
	// VLAN ID set on the interface
	VlanID *uint `json:"vlanID,omitempty"`
	// The interface on which the VLAN was created
	VlanBase string `json:"vlanbase,omitempty"`
	// Link aggregation spec of the interface
	LinkAggregation *LinkAggregationSpec `json:"linkaggregation,omitempty"`
	// Configuration of a bridge connected to the interface
	Bridge *BridgeSpec `json:"bridge,omitempty"`
	// IPv4 configuration of the interface
	IPv4 *IPv4Spec `json:"ipv4,omitempty"`
	// IPv6 configuration of the interface
	IPv6 *IPv6Spec `json:"ipv6,omitempty"`
}

// LinkAggregationSpec aggregation spec of an interface
type LinkAggregationSpec struct {
	// Link aggregation mode
	Mode LinkAggregationMode `json:"mode"`
	// List of slave interfaces aggregated by the interface
	Slaves *SlaveList `json:"slaves,omitempty"`
	// TODO: description
	Options *LinkAggregationOptions `json:"options,omitempty"`
}

// SlaveList is the list of slave interfaces aggregated by the interface
type SlaveList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []string `json:"items,omitempty"`
}

// LinkAggregationOptions TODO: description
type LinkAggregationOptions struct {
	// TODO: description
	Stp *bool `json:"stp,omitempty"`
	// TODO: description
	Rstp *bool `json:"rstp,omitempty"`
	// TODO: description
	FailMode string `json:"failMode,omitempty"`
	// TODO: description
	McastSnoopingEnabled *bool `json:"mcastSnoopingEnabled,omitempty"`
}

// BridgeSpec holds the configuration of the bridge connected to the interface
type BridgeSpec struct {
	// Port configuration on the bridge
	Ports BridgePortList `json:"ports"`
}

// BridgePortList hold the list of ports on the bridge
type BridgePortList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []BridgePort `json:"items"`
}

// BridgePort TODO: description
type BridgePort struct {
	// TODO description
	// TODO: Mandatory?
	Name string `json:"name"`
	// TODO description
	// TODO: Mandatory?
	// TODO: possible values
	Type string `json:"type"`
	// TODO description
	// TODO: Mandatory?
	// TODO: possible values?
	VlanMode string `json:"vlanMode"`
	// TODO description
	// Mandatory?
	AccessTag string `json:"accessTag"`
}

// IPv4Spec hold the IPv4 configuration of the interface
type IPv4Spec struct {
	// Indication whether IPv4 is enabled
	Enabled bool `json:"enabled"`
	// Whether IPv4 addresses are dynamically configured
	DHCP bool `json:"dhcp"`
	// List of IPv4 addresses
	Addresses AddressList `json:"addresses"`
	// TODO: description
	// TODO: Mandatory?
	Neighbors NeighborList `json:"neighbors"`
	// TODO: description
	// TODO: Mandatory?
	Forwarding bool `json:"forwarding"`
}

// AddressList list of CIDRs
type AddressList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []CIDR `json:"items"`
}

// CIDR has IP and prefix length
type CIDR struct {
	IP           string `json:"ip"`
	PrefixLength uint8  `json:"prefixLength"`
}

// NeighborList TODO: add description
type NeighborList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Neighbor `json:"items"`
}

// Neighbor TODO: add description
type Neighbor struct {
	IP               string `json:"ip"`
	LinkLayerAddress string `json:"linkLayerAddress"`
}

// IPv6Spec hold the IPv6 configuration of the interface
type IPv6Spec struct {
	// Indication whether IPv4 is enabled
	Enabled bool `json:"enabled"`
	// Whether IPv4 addresses are dynamically configured
	DHCP bool `json:"dhcp"`
	// IPv6 autoconf
	AutoConf bool `json:"autoConf"`
	// List of IPv4 addresses
	Addresses AddressList `json:"addresses"`
	// TODO: description
	// TODO: Mandatory?
	Neighbors NeighborList `json:"neighbors"`
	// TODO: description
	// TODO: Mandatory?
	Forwarding bool `json:"forwarding"`
	// TODO: description?
	DupAddrDetectTransmit int `json:"dupAddressDetectTransmit"`
}

// CapabilityList TODO: description
type CapabilityList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []string `json:"items,omitempty"`
}

// InterfaceInfoList list of interfaces and their configuration.
type InterfaceInfoList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []InterfaceInfo `json:"items"`
}

// InterfaceInfo holds the operational status of an interface
type InterfaceInfo struct {
	// Current configuration of the interface
	InterfaceSpec `json:",inline"`
	// Interface index
	IFIndex uint `json:"ifIndex"`
	// TODO: list of possible values
	AdminStatus string `json:"adminStatus"`
	// TODO: list of possible values
	LinkStatus string `json:"linkStatus"`
	// Physical address
	// TODO: do we need that on top of the MAC address from InterfaceSpec?
	PhysAddress string `json:"physAddress"`
	// Name of the higher layer interface
	HigherLayerIF string `json:"higherLayerInterface,omitempty"`
	// Name of the lower layer interface
	LowerLayerIF string `json:"lowerLayerInterface,omitempty"`
	// Statistics of the interface
	Statistics InterfaceStatistics `json:"statistics"`
}

// InterfaceStatistics holds the counters on the interface
type InterfaceStatistics struct {
	InBroadcastPackets  uint64 `json:"inBroadcastPackets"`
	InDiscards          uint64 `json:"inDiscards"`
	InErrors            uint64 `json:"inErrors"`
	InMulticastPackets  uint64 `json:"inMulticastPackets"`
	InOctets            uint64 `json:"inOctets"`
	InUnicastPackets    uint64 `json:"inUnicastPackets"`
	OutBroadcastPackets uint64 `json:"outBroadcastPackets"`
	OutDiscards         uint64 `json:"outDiscards"`
	OutErrors           uint64 `json:"outErrors"`
	OutMulticastPackets uint64 `json:"outMulticastPackets"`
	OutOctets           uint64 `json:"outOctets"`
	OutUnicastPackets   uint64 `json:"outUnicastPackets"`
}

// MatchSpec is a list of InterfaceMatchRule
type MatchRules struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []InterfaceMatchRule `json:"items"`
}

// InterfaceMatchRule is a rule describing a match on a specifc interface.
// If multiple parameters are provided, all must match the interface, for it
// to be included in the list of matched interfaces.
type InterfaceMatchRule struct {
	// Name of the interface to match
	Name string `json:"name,omitempty"`
	// Type of the interface to match
	Type InterfaceType `json:"type,omitempty"`
	// VLAN ID on the interface to match
	VlanID *uint `json:"vlanID"`
	// Information retrieved from LLDP on neighbors
	LLDP *LLDPInfo `json:"lldp"`
}

// LLDPInfo holds information retrieved via LLDP protocol from neighboring interfaces
type LLDPInfo struct {
	VlanIDList []uint `json:"vlanIDList,omitempty"`
	// more fields may be added here in the future. E.g chassis-id
}

// AutoConfigSpec defines which autoconfiguration to apply on the interfaces
type AutoConfigSpec struct {
	// AutoBonding indicate whether to do auto bonding on interfaces
	Autobonding *bool `json:"autoBonding"`
	// AutoVlan indicate whether to allocate vlans automatically  to interfaces
	AutoVlan *bool `json:"autoVlan"`
}

// string constants
///////////////////

// InterfaceType is the type of the interface. One of:
// unknown, vlan, ethernet, bond, ovs-bridge
type InterfaceType string

const (
	InterfaceTypeUnknown   = "unknown"
	InterfaceTypeVlan      = "vlan"
	InterfaceTypeEthernet  = "ethernet"
	InterfaceTypeBond      = "bond"
	InterfaceTypeOVSBridge = "ovs-bridge"
	InterfaceTypeDummy     = "dummy"
)

// InterfaceState is the state of the interface. One of:
// absent, up, down, unknown
type InterfaceState string

const (
	InterfaceStateAbsent  = "absent"
	InterfaceStateUp      = "up"
	InterfaceStateDown    = "down"
	InterfaceStateUnknown = "unknown"
)

// DuplexType is either "full" or "half"
type DuplexType string

const (
	DuplexTypeFull = "full"
	DuplexTypeHalf = "half"
)

// LinkAggregationMode one of the following modes:
type LinkAggregationMode string

// TODO: add const strings
