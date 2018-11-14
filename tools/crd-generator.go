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
	"flag"
	"fmt"

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

	crd.ObjectMeta.Name = "nodenetconfpolicies." + v1.SchemeGroupVersionNodeNetConfPolicy.Group
	crd.Spec = extensionsv1.CustomResourceDefinitionSpec{
		Group:   v1.SchemeGroupVersionNodeNetConfPolicy.Group,
		Version: v1.SchemeGroupVersionNodeNetConfPolicy.Version,
		Scope:   "Namespaced",

		Names: extensionsv1.CustomResourceDefinitionNames{
			Plural:   "node-net-conf-policies",
			Singular: "node-net-conf-policy",
			Kind:     v1.SchemeGroupVersionNodeNetConfPolicy.Kind,
			//ShortNames: []string{"net-conf", "net-confs"},
		},
	}

	crdutils.MarshallCrd(crd, "yaml")
}

func generateNodeNetworkStateCrd() {
	crd := generateBlankCrd()

	crd.ObjectMeta.Name = "nodenetworkstates." + v1.SchemeGroupVersionNodeNetworkSate.Group
	crd.Spec = extensionsv1.CustomResourceDefinitionSpec{
		Group:   v1.SchemeGroupVersionNodeNetworkSate.Group,
		Version: v1.SchemeGroupVersionNodeNetworkSate.Version,
		Scope:   "Namespaced",

		Names: extensionsv1.CustomResourceDefinitionNames{
			Plural:   "node-network-states",
			Singular: "node-network-state",
			Kind:     v1.SchemeGroupVersionNodeNetConfPolicy.Kind,
			//ShortNames: []string{"net-state", "net-states"},
		},
	}

	crdutils.MarshallCrd(crd, "yaml")
}

func main() {
	crdType := flag.String("crd-type", "", "Type of crd to generate. net-conf | net-state")
	flag.Parse()

	switch *crdType {
	case "net-conf":
		generateNodeNetConfPolicyCrd()
	case "net-state":
		generateNodeNetworkStateCrd()
	default:
		panic(fmt.Errorf("unknown crd type %s", *crdType))
	}
}
