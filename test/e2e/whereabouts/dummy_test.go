package handler

import (
	"context"
	"net"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/k8snetworkplumbingwg/whereabouts/pkg/storage/kubernetes"
	whereaboutstypes "github.com/k8snetworkplumbingwg/whereabouts/pkg/types"
)

var _ = Describe("Whereabouts", func() {
	It("should reserve and IP address for node", func() {
		ipamConf := whereaboutstypes.IPAMConfig{
			PodNamespace: "kube-system",
			PodName:      "node01",
			Kubernetes: whereaboutstypes.KubernetesConfig{
				KubeConfigPath: os.Getenv("KUBECONFIG"),
			},
			LeaderLeaseDuration: 1500,
			LeaderRenewDeadline: 1000,
			LeaderRetryPeriod:   500,
			Range:               "10.10.10.0/16",
			RangeStart:          net.ParseIP("10.10.10.1"),
			RangeEnd:            net.ParseIP("10.10.10.10"),
		}
		containerID := "eth1"
		podRef := "node01"

		By("Allocating an IP")
		newip, err := kubernetes.IPManagement(context.Background(), whereaboutstypes.Allocate, ipamConf, containerID, podRef)
		Expect(err).ToNot(HaveOccurred())
		Expect(newip.IP).ToNot(BeEmpty())
		Expect(newip.Mask).ToNot(BeEmpty())

		By("Deallocating an IP")
		_, err = kubernetes.IPManagement(context.Background(), whereaboutstypes.Deallocate, ipamConf, containerID, podRef)
		Expect(err).ToNot(HaveOccurred())
	})
})
