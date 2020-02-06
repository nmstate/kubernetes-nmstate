package certificate

import (
	v1beta1 "k8s.io/api/certificates/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	clientv1beta1 "k8s.io/client-go/kubernetes/typed/certificates/v1beta1"
)

type approver struct {
	client clientv1beta1.CertificateSigningRequestInterface
}

func (a approver) approve(csr *v1beta1.CertificateSigningRequest) (*v1beta1.CertificateSigningRequest, error) {
	csr.Status.Conditions = append(csr.Status.Conditions, v1beta1.CertificateSigningRequestCondition{
		Type:    v1beta1.CertificateApproved,
		Reason:  "AutoApproved by kube-admission-webhook",
		Message: "Auto approving webhook server certificate",
	})
	return a.client.UpdateApproval(csr)
}

func newCSRApprover(client clientv1beta1.CertificateSigningRequestInterface) clientv1beta1.CertificateSigningRequestInterface {
	return approver{
		client: client,
	}
}

func (a approver) Create(csr *v1beta1.CertificateSigningRequest) (*v1beta1.CertificateSigningRequest, error) {
	csr, err := a.client.Create(csr)
	if err != nil {
		return csr, err
	}
	return a.approve(csr)
}

func (a approver) Update(csr *v1beta1.CertificateSigningRequest) (*v1beta1.CertificateSigningRequest, error) {
	return a.client.Update(csr)
}

func (a approver) UpdateStatus(csr *v1beta1.CertificateSigningRequest) (*v1beta1.CertificateSigningRequest, error) {
	return a.client.UpdateStatus(csr)
}

func (a approver) Delete(name string, options *v1.DeleteOptions) error {
	return a.client.Delete(name, options)
}

func (a approver) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return a.client.DeleteCollection(options, listOptions)
}

func (a approver) Get(name string, options v1.GetOptions) (*v1beta1.CertificateSigningRequest, error) {
	csr, err := a.client.Get(name, options)
	if err != nil {
		return csr, err
	}
	return a.approve(csr)
}

func (a approver) List(opts v1.ListOptions) (*v1beta1.CertificateSigningRequestList, error) {
	return a.client.List(opts)
}

func (a approver) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return a.client.Watch(opts)
}

func (a approver) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1beta1.CertificateSigningRequest, err error) {
	return a.client.Patch(name, pt, data, subresources...)
}

func (a approver) UpdateApproval(csr *v1beta1.CertificateSigningRequest) (result *v1beta1.CertificateSigningRequest, err error) {
	return a.client.UpdateApproval(csr)
}
