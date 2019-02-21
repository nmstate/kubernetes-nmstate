/*
 * Copyright 2019 Red Hat, Inc.
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
 */

package tests_test

import (
	"flag"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	ginkgo_reporters "kubevirt.io/qe-tools/pkg/ginkgo-reporters"

	nmstate "github.com/nmstate/kubernetes-nmstate/pkg/client/clientset/versioned"
)

var kubeconfig *string
var k8sClientset *kubernetes.Clientset
var nmstateClientset *nmstate.Clientset

func TestPlugin(t *testing.T) {
	RegisterFailHandler(Fail)
	reporters := make([]Reporter, 0)
	if ginkgo_reporters.JunitOutput != "" {
		reporters = append(reporters, ginkgo_reporters.NewJunitReporter())
	}
	RunSpecsWithDefaultAndCustomReporters(t, "kubernetes-nmstate suite", reporters)
}

var _ = BeforeSuite(func() {
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	Expect(err).ToNot(HaveOccurred())
	k8sClientset, err = kubernetes.NewForConfig(config)
	Expect(err).ToNot(HaveOccurred())
	nmstateClientset, err = nmstate.NewForConfig(config)
	Expect(err).ToNot(HaveOccurred())
})

func init() {
	kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	flag.Parse()
}
