package api

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/operator-framework/operator-registry/internal/model"
	"github.com/operator-framework/operator-registry/internal/property"
)

func ConvertAPIBundleToModelBundle(b *Bundle) (*model.Bundle, error) {
	bundleProps, err := convertAPIBundleToModelProperties(b)
	if err != nil {
		return nil, fmt.Errorf("convert properties: %v", err)
	}

	relatedImages, err := getRelatedImages(b.CsvJson)
	if err != nil {
		return nil, fmt.Errorf("get related iamges: %v", err)
	}

	return &model.Bundle{
		Name:          b.CsvName,
		Image:         b.BundlePath,
		Replaces:      b.Replaces,
		Skips:         b.Skips,
		CsvJSON:       b.CsvJson,
		Objects:       b.Object,
		Properties:    bundleProps,
		RelatedImages: relatedImages,
	}, nil
}

func convertAPIBundleToModelProperties(b *Bundle) ([]property.Property, error) {
	var out []property.Property

	for _, skip := range b.Skips {
		out = append(out, property.MustBuildSkips(skip))
	}

	if b.SkipRange != "" {
		out = append(out, property.MustBuildSkipRange(b.SkipRange))
	}

	out = append(out, property.MustBuildChannel(b.ChannelName, b.Replaces))

	providedGVKs := map[property.GVK]struct{}{}
	requiredGVKs := map[property.GVKRequired]struct{}{}

	foundPackageProperty := false
	for i, p := range b.Properties {
		switch p.Type {
		case property.TypeGVK:
			var v GroupVersionKind
			if err := json.Unmarshal(json.RawMessage(p.Value), &v); err != nil {
				return nil, property.ParseError{Idx: i, Typ: p.Type, Err: err}
			}
			k := property.GVK{Group: v.Group, Kind: v.Kind, Version: v.Version}
			providedGVKs[k] = struct{}{}
		case property.TypePackage:
			foundPackageProperty = true
			out = append(out, property.Property{
				Type:  property.TypePackage,
				Value: json.RawMessage(p.Value),
			})
		default:
			out = append(out, property.Property{
				Type:  p.Type,
				Value: json.RawMessage(p.Value),
			})
		}
	}

	for i, p := range b.Dependencies {
		switch p.Type {
		case property.TypeGVK:
			var v GroupVersionKind
			if err := json.Unmarshal(json.RawMessage(p.Value), &v); err != nil {
				return nil, property.ParseError{Idx: i, Typ: p.Type, Err: err}
			}
			k := property.GVKRequired{Group: v.Group, Kind: v.Kind, Version: v.Version}
			requiredGVKs[k] = struct{}{}
		case property.TypePackage:
			var v property.Package
			if err := json.Unmarshal(json.RawMessage(p.Value), &v); err != nil {
				return nil, property.ParseError{Idx: i, Typ: p.Type, Err: err}
			}
			out = append(out, property.MustBuildPackageRequired(v.PackageName, v.Version))
		}
	}

	if !foundPackageProperty {
		out = append(out, property.MustBuildPackage(b.PackageName, b.Version))
	}

	for _, p := range b.ProvidedApis {
		k := property.GVK{Group: p.Group, Kind: p.Kind, Version: p.Version}
		if _, ok := providedGVKs[k]; !ok {
			providedGVKs[k] = struct{}{}
		}
	}
	for _, p := range b.RequiredApis {
		k := property.GVKRequired{Group: p.Group, Kind: p.Kind, Version: p.Version}
		if _, ok := requiredGVKs[k]; !ok {
			requiredGVKs[k] = struct{}{}
		}
	}

	for p := range providedGVKs {
		out = append(out, property.MustBuildGVK(p.Group, p.Version, p.Kind))
	}

	for p := range requiredGVKs {
		out = append(out, property.MustBuildGVKRequired(p.Group, p.Version, p.Kind))
	}

	for i, obj := range b.Object {
		out = append(out, property.MustBuildBundleObjectRef(filepath.Join("objects", b.CsvName, objectFilename(obj, i))))
	}

	return out, nil
}

func getRelatedImages(csvJSON string) ([]model.RelatedImage, error) {
	if len(csvJSON) == 0 {
		return nil, nil
	}
	type csv struct {
		Spec struct {
			RelatedImages []struct {
				Name  string `json:"name"`
				Image string `json:"image"`
			} `json:"relatedImages"`
		} `json:"spec"`
	}
	c := csv{}
	if err := json.Unmarshal([]byte(csvJSON), &c); err != nil {
		return nil, fmt.Errorf("unmarshal csv: %v", err)
	}
	relatedImages := []model.RelatedImage{}
	for _, ri := range c.Spec.RelatedImages {
		relatedImages = append(relatedImages, model.RelatedImage(ri))
	}
	return relatedImages, nil
}

func objectFilename(obj string, idx int) string {
	name, kind := fmt.Sprintf("obj%04d", idx), ""
	u := unstructured.Unstructured{}
	if err := yaml.Unmarshal([]byte(obj), &u); err == nil {
		if u.GetName() != "" {
			name = u.GetName()
		}
		gvk := u.GroupVersionKind()
		kind = fmt.Sprintf("%s_%s_%s", gvk.Group, gvk.Version, strings.ToLower(gvk.Kind))
	}
	if kind == "" {
		return fmt.Sprintf("%s", name)
	}
	return fmt.Sprintf("%s_%s.yaml", name, kind)
}
