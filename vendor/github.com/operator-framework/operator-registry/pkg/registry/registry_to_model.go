package registry

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/operator-framework/operator-registry/internal/model"
	"github.com/operator-framework/operator-registry/internal/property"
)

func ConvertRegistryBundleToModelBundles(b *Bundle) ([]model.Bundle, error) {
	var bundles []model.Bundle
	desc, err := b.csv.GetDescription()
	if err != nil {
		return nil, fmt.Errorf("Could not get description from bundle CSV:%s", err)
	}

	i, err := b.csv.GetIcons()
	if err != nil {
		return nil, fmt.Errorf("Could not get icon from bundle CSV:%s", err)
	}
	mIcon := &model.Icon{
		MediaType: "",
		Data:      []byte{},
	}
	if len(i) > 0 {
		mIcon.MediaType = i[0].MediaType
		mIcon.Data = []byte(i[0].Base64data)
	}

	pkg := &model.Package{
		Name:        b.Annotations.PackageName,
		Description: desc,
		Icon:        mIcon,
		Channels:    make(map[string]*model.Channel),
	}

	mb, err := registryBundleToModelBundle(b)
	mb.Package = pkg
	if err != nil {
		return nil, err
	}

	for _, ch := range extractChannels(b.Annotations.Channels) {
		newCh := &model.Channel{
			Name: ch,
		}
		chBundle := mb
		chBundle.Channel = newCh
		bundles = append(bundles, *chBundle)
	}
	return bundles, nil
}

func registryBundleToModelBundle(b *Bundle) (*model.Bundle, error) {
	bundleProps, err := convertRegistryBundleToModelProperties(b)
	if err != nil {
		return nil, fmt.Errorf("error converting properties for internal model: %v", err)
	}

	csv, err := b.ClusterServiceVersion()
	if err != nil {
		return nil, fmt.Errorf("Could not get CVS for bundle: %s", err)
	}
	replaces, err := csv.GetReplaces()
	if err != nil {
		return nil, fmt.Errorf("Could not get Replaces from CSV for bundle: %s", err)
	}
	skips, err := csv.GetSkips()
	if err != nil {
		return nil, fmt.Errorf("Could not get Skips from CSV for bundle: %s", err)
	}
	relatedImages, err := convertToModelRelatedImages(csv)
	if err != nil {
		return nil, fmt.Errorf("Could not get Related images from bundle: %v", err)
	}

	return &model.Bundle{
		Name:          csv.Name,
		Image:         b.BundleImage,
		Replaces:      replaces,
		Skips:         skips,
		Properties:    bundleProps,
		RelatedImages: relatedImages,
	}, nil
}

func convertRegistryBundleToModelProperties(b *Bundle) ([]property.Property, error) {
	var out []property.Property

	skips, err := b.csv.GetSkips()
	if err != nil {
		return nil, fmt.Errorf("Could not get Skips from CSV for bundle: %s", err)
	}

	for _, skip := range skips {
		out = append(out, property.MustBuildSkips(skip))
	}

	skipRange := b.csv.GetSkipRange()
	if skipRange != "" {
		out = append(out, property.MustBuildSkipRange(skipRange))
	}

	replaces, err := b.csv.GetReplaces()
	if err != nil {
		return nil, fmt.Errorf("Could not get Replaces from CSV for bundle: %s", err)
	}
	for _, ch := range extractChannels(b.Annotations.Channels) {
		out = append(out, property.MustBuildChannel(ch, replaces))
	}

	providedGVKs := map[property.GVK]struct{}{}
	requiredGVKs := map[property.GVKRequired]struct{}{}

	foundPackageProperty := false
	for i, p := range b.Properties {
		switch p.Type {
		case property.TypeGVK:
			var v property.GVK
			if err := json.Unmarshal(json.RawMessage(p.Value), &v); err != nil {
				return nil, property.ParseError{Idx: i, Typ: p.Type, Err: err}
			}
			k := property.GVK{Group: v.Group, Kind: v.Kind, Version: v.Version}
			providedGVKs[k] = struct{}{}
		case property.TypePackage:
			var v property.Package
			if err := json.Unmarshal(json.RawMessage(p.Value), &v); err != nil {
				return nil, property.ParseError{Idx: i, Typ: p.Type, Err: err}
			}
			out = append(out, property.MustBuildPackageRequired(v.PackageName, v.Version))
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
			var v property.GVK
			if err := json.Unmarshal(json.RawMessage(p.Value), &v); err != nil {
				return nil, property.ParseError{Idx: i, Typ: p.Type, Err: err}
			}
			k := property.GVKRequired{Group: v.Group, Kind: v.Kind, Version: v.Version}
			requiredGVKs[k] = struct{}{}
		case property.TypePackage:
			out = append(out, property.Property{
				Type:  property.TypePackageRequired,
				Value: json.RawMessage(p.Value),
			})
		}
	}

	version, err := b.Version()
	if err != nil {
		return nil, fmt.Errorf("error getting bundle version from CSV %q:%v", b.csv.Name, err)
	}

	if !foundPackageProperty {
		out = append(out, property.MustBuildPackage(b.Annotations.PackageName, version))
	}

	providedApis, err := b.ProvidedAPIs()
	if err != nil {
		return nil, fmt.Errorf("error getting Provided APIs for bundle %q:%v", b.Name, err)
	}

	for p := range providedApis {
		k := property.GVK{Group: p.Group, Kind: p.Kind, Version: p.Version}
		if _, ok := providedGVKs[k]; !ok {
			providedGVKs[k] = struct{}{}
		}
	}
	requiredApis, err := b.RequiredAPIs()
	if err != nil {
		return nil, fmt.Errorf("Could not get Required APIs from bundle:%s", err)
	}
	for p := range requiredApis {
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

	return out, nil
}

func convertToModelRelatedImages(csv *ClusterServiceVersion) ([]model.RelatedImage, error) {
	var objmap map[string]*json.RawMessage
	if err := json.Unmarshal(csv.Spec, &objmap); err != nil {
		return nil, err
	}

	rawValue, ok := objmap[relatedImages]
	if !ok || rawValue == nil {
		return nil, nil
	}

	type relatedImage struct {
		Name string `json:"name"`
		Ref  string `json:"image"`
	}
	var relatedImages []relatedImage
	if err := json.Unmarshal(*rawValue, &relatedImages); err != nil {
		return nil, err
	}
	mrelatedImages := []model.RelatedImage{}
	for _, img := range relatedImages {
		mrelatedImages = append(mrelatedImages, model.RelatedImage{Name: img.Name, Image: img.Ref})
	}
	return mrelatedImages, nil
}

func extractChannels(annotationChannels string) []string {
	var channels []string
	for _, ch := range strings.Split(annotationChannels, ",") {
		c := strings.TrimSpace(ch)
		if c != "" {
			channels = append(channels, ch)
		}
	}
	return channels
}
