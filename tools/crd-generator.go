/*
 * This file is part of the nmstate project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2018 Red Hat, Inc.
 *
 */

package main

import (
	"encoding/json"
	"flag"
	"fmt"

	yamlutils "github.com/ghodss/yaml"

	crdutils "github.com/ant31/crd-validation/pkg"
	extensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/nmstate/k8s-node-net-conf/pkg/apis/nmstate.io/v1"
)

func generateBlankCrd() *extensionsv1.CustomResourceDefinition {
	return &extensionsv1.CustomResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apiextensions.k8s.io/v1beta1",
			Kind:       "CustomResourceDefinition",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"nmstate.io": "",
			},
		},
	}
}

func generateNodeNetConfPolicyCrd() {
	crd := generateBlankCrd()

	pluralForm := "nodenetconfpolicies"
	crd.ObjectMeta.Name = pluralForm + "." + v1.SchemeGroupVersionNodeNetConfPolicy.Group
	crd.Spec = extensionsv1.CustomResourceDefinitionSpec{
		Group:   v1.SchemeGroupVersionNodeNetConfPolicy.Group,
		Version: v1.SchemeGroupVersionNodeNetConfPolicy.Version,
		Scope:   "Namespaced",

		Names: extensionsv1.CustomResourceDefinitionNames{
			Plural:     pluralForm,
			Singular:   "nodenetconfpolicy",
			Kind:       v1.SchemeGroupVersionNodeNetConfPolicy.Kind,
			ShortNames: []string{"net-conf", "net-confs"},
		},
	}

	crdutils.MarshallCrd(crd, "yaml")
}

func generateNodeNetworkStateCrd() {
	crd := generateBlankCrd()

	pluralForm := "nodenetworkstates"
	crd.ObjectMeta.Name = pluralForm + "." + v1.SchemeGroupVersionNodeNetworkState.Group
	crd.Spec = extensionsv1.CustomResourceDefinitionSpec{
		Group:   v1.SchemeGroupVersionNodeNetworkState.Group,
		Version: v1.SchemeGroupVersionNodeNetworkState.Version,
		Scope:   "Namespaced",

		Names: extensionsv1.CustomResourceDefinitionNames{
			Plural:     pluralForm,
			Singular:   "nodenetworkstate",
			Kind:       v1.SchemeGroupVersionNodeNetworkState.Kind,
			ShortNames: []string{"net-state", "net-states"},
		},
	}

	crdutils.MarshallCrd(crd, "yaml")
}

const (
	JsonPrefix = ""
	JsonIndent = "    "
)

func generateNodeNetConfPolicySample() {
	sample := v1.NodeNetConfPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1.SchemeGroupVersionNodeNetConfPolicy.Kind,
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-node-net-conf-policy",
		},
		Spec: v1.NodeNetConfPolicySpec{},
	}

	json, err := json.MarshalIndent(sample, JsonPrefix, JsonIndent)
	if err != nil {
		panic(fmt.Errorf("failed to generate sample (json): %v", err))
	}

	yaml, err := yamlutils.JSONToYAML(json)
	if err != nil {
		panic(fmt.Errorf("failed to generate sample (yaml): %v", err))
	}
	fmt.Println(string(yaml))
}

// based on: https://nmstate.github.io/examples.html
func generateNodeNetworkStateSample() {
	MTU1400 := uint(1400)
	AutoNegotiate := true

	sample := v1.NodeNetworkState{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1.SchemeGroupVersionNodeNetworkState.Kind,
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "node1-network-state",
		},
		Spec: v1.NodeNetworkStateSpec{
			Managed:  true,
			NodeName: "node1",
			DesiredState: v1.ConfigurationState{
				Interfaces: []v1.InterfaceSpec{
					{
						// setting ethernet interface with static IPs
						Name: "eth0", Type: v1.InterfaceTypeEthernet, State: v1.InterfaceStateUp, MTU: &MTU1400,
						Description: "Production Network", AutoNegotiation: &AutoNegotiate, Duplex: v1.DuplexTypeFull,
						IPv4: &v1.IPv4Spec{
							Enabled:   true,
							DHCP:      false,
							Addresses: []v1.CIDR{{IP: "10.0.0.2", PrefixLength: 24}},
							Neighbors: []v1.Neighbor{{IP: "10.0.0.1", LinkLayerAddress: "00:25:96:FF:FE:12:34:56"}},
						},
						IPv6: &v1.IPv6Spec{
							Enabled:   true,
							DHCP:      false,
							Addresses: []v1.CIDR{{IP: "2001:db8::1:1", PrefixLength: 64}},
						},
					},
					{
						// setting ethernet interface with DHCP
						Name: "eth1", Type: v1.InterfaceTypeEthernet, State: v1.InterfaceStateUp, MTU: &MTU1400,
						Description: "Production Network", AutoNegotiation: &AutoNegotiate, Duplex: v1.DuplexTypeFull,
						IPv4: &v1.IPv4Spec{
							Enabled: true,
							DHCP:    true,
						},
						IPv6: &v1.IPv6Spec{
							Enabled: true,
							DHCP:    true,
						},
					},
					{
						// setting interface down
						Name: "old-br", Type: v1.InterfaceTypeOVSBridge, State: v1.InterfaceStateDown,
						Description: "Deprecated Bridge",
					},
					{
						// removing an interface
						Name: "dummy0", Type: v1.InterfaceTypeDummy, State: v1.InterfaceStateAbsent,
						Description: "Another Deprecated Bridge",
					},
				},
			},
		},
		Status: v1.NodeNetworkStateStatus{},
	}

	json, err := json.MarshalIndent(sample, JsonPrefix, JsonIndent)
	if err != nil {
		panic(fmt.Errorf("failed to generate sample (json): %v", err))
	}

	yaml, err := yamlutils.JSONToYAML(json)
	if err != nil {
		panic(fmt.Errorf("failed to generate sample (yaml): %v", err))
	}
	fmt.Println(string(yaml))
}

func main() {
	crdType := flag.String("crd-type", "", "Type of crd to generate. net-conf | net-state | net-conf-sample | net-state-sample")
	flag.Parse()

	switch *crdType {
	case "net-conf":
		generateNodeNetConfPolicyCrd()
	case "net-state":
		generateNodeNetworkStateCrd()
	case "net-conf-sample":
		generateNodeNetConfPolicySample()
	case "net-state-sample":
		generateNodeNetworkStateSample()
	default:
		panic(fmt.Errorf("unknown crd type %s", *crdType))
	}
}
