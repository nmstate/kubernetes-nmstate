package v1beta1

import (
	"sigs.k8s.io/controller-runtime/pkg/conversion"

	"github.com/nmstate/kubernetes-nmstate/api/v1alpha1"
)

func (src *NodeNetworkConfigurationPolicy) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1alpha1.NodeNetworkConfigurationPolicy)
	dst.ObjectMeta = src.ObjectMeta
	dst.Spec = src.Spec
	dst.Status = src.Status
	return nil
}

func (dst *NodeNetworkConfigurationPolicy) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1alpha1.NodeNetworkConfigurationPolicy)
	dst.ObjectMeta = src.ObjectMeta
	dst.Spec = src.Spec
	dst.Status = src.Status
	return nil
}
