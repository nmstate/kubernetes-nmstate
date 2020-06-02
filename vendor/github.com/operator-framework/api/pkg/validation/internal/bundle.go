package internal

import (
	"fmt"

	"github.com/operator-framework/api/pkg/manifests"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/api/pkg/validation/errors"
	interfaces "github.com/operator-framework/api/pkg/validation/interfaces"
)

var BundleValidator interfaces.Validator = interfaces.ValidatorFunc(validateBundles)

func validateBundles(objs ...interface{}) (results []errors.ManifestResult) {
	for _, obj := range objs {
		switch v := obj.(type) {
		case *manifests.Bundle:
			results = append(results, validateBundle(v))
		}
	}
	return results
}

func validateBundle(bundle *manifests.Bundle) (result errors.ManifestResult) {
	result = validateOwnedCRDs(bundle, bundle.CSV)
	result.Name = bundle.CSV.Spec.Version.String()
	return result
}

func validateOwnedCRDs(bundle *manifests.Bundle, csv *operatorsv1alpha1.ClusterServiceVersion) (result errors.ManifestResult) {
	ownedCrdNames := getOwnedCustomResourceDefintionNames(csv)
	crdNames, err := getBundleCRDNames(bundle)
	if err != (errors.Error{}) {
		result.Add(err)
		return result
	}

	// validating names
	for _, crdName := range ownedCrdNames {
		if _, ok := crdNames[crdName]; !ok {
			result.Add(errors.ErrInvalidBundle(fmt.Sprintf("owned CRD %q not found in bundle %q", crdName, bundle.Name), crdName))
		} else {
			delete(crdNames, crdName)
		}
	}
	// CRDs not defined in the CSV present in the bundle
	for crdName := range crdNames {
		result.Add(errors.WarnInvalidBundle(fmt.Sprintf("owned CRD %q is present in bundle %q but not defined in CSV", crdName, bundle.Name), crdName))
	}
	return result
}

func getOwnedCustomResourceDefintionNames(csv *operatorsv1alpha1.ClusterServiceVersion) (names []string) {
	for _, ownedCrd := range csv.Spec.CustomResourceDefinitions.Owned {
		names = append(names, ownedCrd.Name)
	}
	return names
}

func getBundleCRDNames(bundle *manifests.Bundle) (map[string]struct{}, errors.Error) {
	crdNames := map[string]struct{}{}
	for _, crd := range bundle.V1beta1CRDs {
		crdNames[crd.GetName()] = struct{}{}
	}
	return crdNames, errors.Error{}
}
