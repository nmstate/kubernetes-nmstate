package apply

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	uns "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ApplyObject applies the desired object against the apiserver,
// merging it with any existing objects if already present.
func ApplyObject(ctx context.Context, client k8sclient.Client, obj *uns.Unstructured) error {
	name := obj.GetName()
	namespace := obj.GetNamespace()
	if name == "" {
		return errors.Errorf("object %s has no name", obj.GroupVersionKind().String())
	}
	gvk := obj.GroupVersionKind()
	// used for logging and errors
	objDesc := fmt.Sprintf("(%s) %s/%s", gvk.String(), namespace, name)
	log.Printf("reconciling %s", objDesc)

	if err := IsObjectSupported(obj); err != nil {
		return errors.Wrapf(err, "object %s unsupported", objDesc)
	}

	// Get existing
	existing := &uns.Unstructured{}
	existing.SetGroupVersionKind(gvk)
	err := client.Get(ctx, types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}, existing)

	if err != nil && apierrors.IsNotFound(err) {
		log.Printf("does not exist, creating %s", objDesc)
		err := client.Create(ctx, obj)
		if err != nil {
			return errors.Wrapf(err, "could not create %s", objDesc)
		}
		log.Printf("successfully created %s", objDesc)
		return nil
	}
	if err != nil {
		return errors.Wrapf(err, "could not retrieve existing %s", objDesc)
	}

	if isTLSSecret(obj) {
		log.Printf("Ignoring TLS secret %s at reconcile", obj.GetName())
		return nil
	}

	// Merge the desired object with what actually exists
	if err := MergeObjectForUpdate(existing, obj); err != nil {
		return errors.Wrapf(err, "could not merge object %s with existing", objDesc)
	}
	if !equality.Semantic.DeepEqual(existing, obj) {
		if err := client.Update(ctx, obj); err != nil {
			// In older versions of the operator, we used daemon sets of type 'extensions/v1beta1', later we
			// changed that to 'apps/v1'. Because of this change, we are not able to seamlessly upgrade using
			// only Update methods. Following code handles this exception by deleting the old daemon set and
			// creating a new one.
			// TODO: Upgrade transaction should be handled by each component module separately. Once we make
			// that possible, this exception should be dropped.
			bridgeMarkerDaemonSetUpdateError := "DaemonSet.apps \"bridge-marker\" is invalid: spec.selector: Invalid value: v1.LabelSelector{MatchLabels:map[string]string{\"name\":\"bridge-marker\"}, MatchExpressions:[]v1.LabelSelectorRequirement(nil)}: field is immutable"
			if strings.Contains(err.Error(), bridgeMarkerDaemonSetUpdateError) {
				log.Print("update failed due to change in DaemonSet API group; removing original object and recreating")
				if err := client.Delete(ctx, existing); err != nil {
					return errors.Wrapf(err, "could not delete %s", objDesc)
				}
				if err := client.Create(ctx, obj); err != nil {
					return errors.Wrapf(err, "could not create %s", objDesc)
				}
				log.Print("update of conflicting DaemonSet was successful")
			}

			return errors.Wrapf(err, "could not update object %s", objDesc)
		}
		log.Print("update was successful")
	}

	return nil
}

// DeleteOwnedObject deletes an object in the apiserver
func DeleteOwnedObject(ctx context.Context, client k8sclient.Client, obj *uns.Unstructured) error {
	name := obj.GetName()
	namespace := obj.GetNamespace()
	if name == "" {
		return errors.Errorf("object %s has no name", obj.GroupVersionKind().String())
	}

	gvk := obj.GroupVersionKind()
	// used for logging and errors
	objDesc := fmt.Sprintf("(%s) %s/%s", gvk.String(), namespace, name)

	// Get existing
	existing := &uns.Unstructured{}
	existing.SetGroupVersionKind(gvk)
	err := client.Get(ctx, types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}, existing)
	if err != nil {
		// Fail only if the error is not one of the followings:
		// - not found (nothing to do since it's already gone)
		// - the type does not exist (if the resource type does not exist then there is no resource at all)
		if !apierrors.IsNotFound(err) && !apimeta.IsNoMatchError(err) {
			return errors.Wrapf(err, "failed retrieving owned %s", objDesc)
		}
		return nil
	}

	if !cnaoOwns(existing) {
		return nil
	}
	log.Printf("Handling deletion of %s", objDesc)
	if err := client.Delete(ctx, existing); err != nil {
		return errors.Wrapf(err, "failed deleting owned %s", objDesc)
	}

	return nil
}

func cnaoOwns(obj *uns.Unstructured) bool {
	owners := obj.GetOwnerReferences()
	for _, owner := range owners {
		if owner.Kind == "NetworkAddonsConfig" {
			return true
		}
	}
	return false
}

func isTLSSecret(obj *uns.Unstructured) bool {
	if obj.GetKind() != "Secret" {
		return false
	}

	typez, found, err := uns.NestedString(obj.Object, "type")
	if err != nil || !found {
		return false
	}

	return typez == "kubernetes.io/tls"
}
