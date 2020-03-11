package e2e

import (
	"context"
	"fmt"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	testFilter    = "eth1*"
	defaultFilter = "veth*"
	configMapName = "nmstate-config"
)

var _ = Describe("Configurations test", func() {
	Context("Verifying config map creation, deletion and editing", func() {
		BeforeEach(func() {
			By(fmt.Sprintf("Verifying %s is in current state", firstSecondaryNic))
			for _, node := range nodes {
				interfacesNameForNodeEventually(node).Should(ContainElement(firstSecondaryNic))
			}
		})
		AfterEach(func() {
			_ = deleteConfigMap(configMapName, framework.Global.Namespace)
		})

		It("should have NodeNetworkState with currentState for each node", func() {
			err := createConfigmap(configMapName, framework.Global.Namespace, testFilter)
			Expect(err).ShouldNot(HaveOccurred())
			By(fmt.Sprintf("Verifying %s is not in current state", firstSecondaryNic))
			for _, node := range nodes {
				interfacesNameForNodeEventually(node).ShouldNot(ContainElement(firstSecondaryNic))
			}

			By("Returning to default values")
			err = editConfigmap(configMapName, framework.Global.Namespace, defaultFilter)
			Expect(err).ShouldNot(HaveOccurred())

			By(fmt.Sprintf("Verifying %s is in current state", firstSecondaryNic))
			for _, node := range nodes {
				interfacesNameForNodeEventually(node).Should(ContainElement(firstSecondaryNic))
			}
		})
	})
})

func deleteConfigMap(configMapName string, namespace string) error {
	By(fmt.Sprintf("Deleting configmap %s for namespace %s", configMapName, namespace))

	configMap := &corev1.ConfigMap{}
	configMap.Name = configMapName
	configMap.Namespace = namespace

	err := framework.Global.Client.Delete(context.TODO(), configMap)
	if err != nil {
		fmt.Println("Error while deleting configmap ", err.Error())
	}
	return err
}

func createConfigmap(configMapName string, namespace string, filter string) error {
	By(fmt.Sprintf("Creating configmap %s for namespace %s", configMapName, namespace))

	configMap := &corev1.ConfigMap{}
	configMap.Name = configMapName
	configMap.Namespace = namespace
	configMap.Data = map[string]string{"nmstate.yaml": fmt.Sprintf(`"interfaces_filter": "%s"`, filter)}

	err := framework.Global.Client.Create(context.TODO(), configMap, &framework.CleanupOptions{})
	if err != nil {
		fmt.Println("Error while creating configmap ", err.Error())
	}
	return err
}

func editConfigmap(configMapName string, namespace string, filter string) error {
	By(fmt.Sprintf("Editing configmap %s for namespace %s", configMapName, namespace))

	configMap := &corev1.ConfigMap{}
	err := framework.Global.Client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: configMapName}, configMap)
	if err != nil {
		fmt.Println("Error while updating configmap ", err.Error())
	}
	Expect(err).ShouldNot(HaveOccurred())
	configMap.Data = map[string]string{"nmstate.yaml": fmt.Sprintf(`"interfaces_filter": "%s"`, filter)}

	err = framework.Global.Client.Update(context.TODO(), configMap)
	if err != nil {
		fmt.Println("Error while updating configmap ", err.Error())
	}
	return err
}
