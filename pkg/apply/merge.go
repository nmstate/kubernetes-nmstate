package apply

import (
	"github.com/pkg/errors"

	unstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// MergeMetadataForUpdate merges the read-only fields of metadata.
// This is to be able to do a a meaningful comparison in apply,
// since objects created on runtime do not have these fields populated.
func MergeMetadataForUpdate(current, updated *unstructured.Unstructured) error {
	updated.SetCreationTimestamp(current.GetCreationTimestamp())
	updated.SetSelfLink(current.GetSelfLink())
	updated.SetGeneration(current.GetGeneration())
	updated.SetUID(current.GetUID())
	updated.SetResourceVersion(current.GetResourceVersion())

	mergeAnnotations(current, updated)
	mergeLabels(current, updated)

	return nil
}

// MergeObjectForUpdate prepares a "desired" object to be updated.
// Some objects, such as Deployments and Services require
// some semantic-aware updates
func MergeObjectForUpdate(current, updated *unstructured.Unstructured) error {
	if err := MergeDeploymentForUpdate(current, updated); err != nil {
		return err
	}

	if err := MergeServiceForUpdate(current, updated); err != nil {
		return err
	}

	if err := MergeServiceAccountForUpdate(current, updated); err != nil {
		return err
	}

	if err := mergeWebhookConfiguration(current, updated); err != nil {
		return err
	}

	// For all object types, merge metadata.
	// Run this last, in case any of the more specific merge logic has
	// changed "updated"
	MergeMetadataForUpdate(current, updated)

	return nil
}

const (
	deploymentRevisionAnnotation = "deployment.kubernetes.io/revision"
)

// MergeDeploymentForUpdate updates Deployment objects.
// We merge annotations, keeping ours except the Deployment Revision annotation.
func MergeDeploymentForUpdate(current, updated *unstructured.Unstructured) error {
	gvk := updated.GroupVersionKind()
	if gvk.Group == "apps" && gvk.Kind == "Deployment" {

		// Copy over the revision annotation from current up to updated
		// otherwise, updated would win, and this annotation is "special" and
		// needs to be preserved
		curAnnotations := current.GetAnnotations()
		updatedAnnotations := updated.GetAnnotations()
		if updatedAnnotations == nil {
			updatedAnnotations = map[string]string{}
		}

		anno, ok := curAnnotations[deploymentRevisionAnnotation]
		if ok {
			updatedAnnotations[deploymentRevisionAnnotation] = anno
		}

		updated.SetAnnotations(updatedAnnotations)
	}

	return nil
}

// MergeServiceForUpdate ensures the clusterip is never written to
func MergeServiceForUpdate(current, updated *unstructured.Unstructured) error {
	gvk := updated.GroupVersionKind()
	if gvk.Group == "" && gvk.Kind == "Service" {
		clusterIP, found, err := unstructured.NestedString(current.Object, "spec", "clusterIP")
		if err != nil {
			return err
		}

		if found {
			return unstructured.SetNestedField(updated.Object, clusterIP, "spec", "clusterIP")
		}
	}

	return nil
}

// MergeServiceAccountForUpdate copies secrets from current to updated.
// This is intended to preserve the auto-generated token.
// Right now, we just copy current to updated and don't support supplying
// any secrets ourselves.
func MergeServiceAccountForUpdate(current, updated *unstructured.Unstructured) error {
	gvk := updated.GroupVersionKind()
	if gvk.Group == "" && gvk.Kind == "ServiceAccount" {
		curSecrets, ok, err := unstructured.NestedSlice(current.Object, "secrets")
		if err != nil {
			return err
		}

		if ok {
			unstructured.SetNestedField(updated.Object, curSecrets, "secrets")
		}
	}
	return nil
}

func indexWebhooksByName(configuration *unstructured.Unstructured) (map[string]map[string]interface{}, error) {
	webhooks, found, err := unstructured.NestedSlice(configuration.Object, "webhooks")
	if err != nil {
		return nil, errors.Wrap(err, "failed searching for 'webhooks' field at configuration")
	}
	if !found {
		return nil, nil
	}

	webhooksByName := map[string]map[string]interface{}{}
	for _, webhook := range webhooks {
		webhookFields := webhook.(map[string]interface{})
		name, found, err := unstructured.NestedString(webhookFields, "name")
		if err != nil {
			return nil, errors.Wrap(err, "failed searching for 'name' field at webhook")
		}
		if !found {
			continue
		}
		webhooksByName[name] = webhookFields
	}
	return webhooksByName, nil
}

// mergeWebhookConfiguration ensure caBundle is kept at webhooks's clientConfig
func mergeWebhookConfiguration(current, updated *unstructured.Unstructured) error {
	gvk := updated.GroupVersionKind()
	if gvk.Kind != "MutatingWebhookConfiguration" && gvk.Kind != "ValidatingWebhookConfiguration" {
		return nil
	}

	// Keep current webhooks in a map by their names for easier access
	currentWebhooksByName, err := indexWebhooksByName(current)
	if err != nil {
		return errors.Wrap(err, "failed indexing current configuration webhooks by name")
	}

	// Read the list of the newly set webhooks
	updatedWebhooks, found, err := unstructured.NestedSlice(updated.Object, "webhooks")
	if err != nil {
		return errors.Wrap(err, "failed searching for 'webhooks' field at configuration")
	}
	if !found {
		return nil
	}

	// Merge values we want to preserve from the current webhooks into the new ones
	mergedWebhooks := []interface{}{}
	for _, updatedWebhook := range updatedWebhooks {
		updatedWebhookName, found, err := unstructured.NestedString(updatedWebhook.(map[string]interface{}), "name")
		if err != nil {
			return errors.Wrapf(err, "failed reading 'name' in webhook config")
		}
		if !found {
			continue
		}

		currentWebhook, found := currentWebhooksByName[updatedWebhookName]
		if !found {
			continue
		}

		currentCABundle, currentCABundleFound, err := unstructured.NestedString(currentWebhook, "clientConfig", "caBundle")
		if err != nil {
			return errors.Wrapf(err, "failed searching current caBundle 'field' at webhook %s", updatedWebhookName)
		}

		_, updatedCABundleFound, err := unstructured.NestedString(updatedWebhook.(map[string]interface{}), "clientConfig", "caBundle")
		if err != nil {
			return errors.Wrapf(err, "failed searching updated caBundle 'field' at webhook %s", updatedWebhookName)
		}

		// If there is a CABundle field at current configuration and there is no CABundle at
		// updated configuration copy it from current to updated.
		if currentCABundleFound && !updatedCABundleFound {
			err = unstructured.SetNestedField(updatedWebhook.(map[string]interface{}), currentCABundle, "clientConfig", "caBundle")
			if err != nil {
				return errors.Wrapf(err, "failed copying caBundle from current config to updated config at webhook %s", updatedWebhookName)
			}
		}
		mergedWebhooks = append(mergedWebhooks, updatedWebhook)
	}

	err = unstructured.SetNestedSlice(updated.Object, mergedWebhooks, "webhooks")
	if err != nil {
		return errors.Wrap(err, "failed changing 'webhooks' field at updated configuration")
	}

	return nil
}

// mergeAnnotations copies over any annotations from current to updated,
// with updated winning if there's a conflict
func mergeAnnotations(current, updated *unstructured.Unstructured) {
	updatedAnnotations := updated.GetAnnotations()
	curAnnotations := current.GetAnnotations()

	if curAnnotations == nil {
		curAnnotations = map[string]string{}
	}

	for k, v := range updatedAnnotations {
		curAnnotations[k] = v
	}

	updated.SetAnnotations(curAnnotations)
}

// mergeLabels copies over any labels from current to updated,
// with updated winning if there's a conflict
func mergeLabels(current, updated *unstructured.Unstructured) {
	updatedLabels := updated.GetLabels()
	curLabels := current.GetLabels()

	if curLabels == nil {
		curLabels = map[string]string{}
	}

	for k, v := range updatedLabels {
		curLabels[k] = v
	}

	updated.SetLabels(curLabels)
}

// IsObjectSupported rejects objects with configurations we don't support.
// This catches ServiceAccounts with secrets, which is valid but we don't
// support reconciling them.
func IsObjectSupported(obj *unstructured.Unstructured) error {
	gvk := obj.GroupVersionKind()

	// We cannot create ServiceAccounts with secrets because there's currently
	// no need and the merging logic is complex.
	// If you need this, please file an issue.
	if gvk.Group == "" && gvk.Kind == "ServiceAccount" {
		secrets, ok, err := unstructured.NestedSlice(obj.Object, "secrets")
		if err != nil {
			return err
		}

		if ok && len(secrets) > 0 {
			return errors.Errorf("cannot create ServiceAccount with secrets")
		}
	}

	return nil
}
