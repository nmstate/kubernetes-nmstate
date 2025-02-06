/*
Copyright The Kubernetes Authors.

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

// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"

	v1alpha1 "k8s.io/api/networking/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	networkingv1alpha1 "k8s.io/client-go/applyconfigurations/networking/v1alpha1"
	gentype "k8s.io/client-go/gentype"
	scheme "k8s.io/client-go/kubernetes/scheme"
)

// IPAddressesGetter has a method to return a IPAddressInterface.
// A group's client should implement this interface.
type IPAddressesGetter interface {
	IPAddresses() IPAddressInterface
}

// IPAddressInterface has methods to work with IPAddress resources.
type IPAddressInterface interface {
	Create(ctx context.Context, iPAddress *v1alpha1.IPAddress, opts v1.CreateOptions) (*v1alpha1.IPAddress, error)
	Update(ctx context.Context, iPAddress *v1alpha1.IPAddress, opts v1.UpdateOptions) (*v1alpha1.IPAddress, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.IPAddress, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.IPAddressList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.IPAddress, err error)
	Apply(ctx context.Context, iPAddress *networkingv1alpha1.IPAddressApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.IPAddress, err error)
	IPAddressExpansion
}

// iPAddresses implements IPAddressInterface
type iPAddresses struct {
	*gentype.ClientWithListAndApply[*v1alpha1.IPAddress, *v1alpha1.IPAddressList, *networkingv1alpha1.IPAddressApplyConfiguration]
}

// newIPAddresses returns a IPAddresses
func newIPAddresses(c *NetworkingV1alpha1Client) *iPAddresses {
	return &iPAddresses{
		gentype.NewClientWithListAndApply[*v1alpha1.IPAddress, *v1alpha1.IPAddressList, *networkingv1alpha1.IPAddressApplyConfiguration](
			"ipaddresses",
			c.RESTClient(),
			scheme.ParameterCodec,
			"",
			func() *v1alpha1.IPAddress { return &v1alpha1.IPAddress{} },
			func() *v1alpha1.IPAddressList { return &v1alpha1.IPAddressList{} }),
	}
}
