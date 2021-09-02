package declcfg

import (
	"sort"

	"github.com/operator-framework/operator-registry/internal/model"
	"github.com/operator-framework/operator-registry/internal/property"
)

func ConvertFromModel(mpkgs model.Model) DeclarativeConfig {
	cfg := DeclarativeConfig{}
	for _, mpkg := range mpkgs {
		bundles := traverseModelChannels(*mpkg)

		var i *Icon
		if mpkg.Icon != nil {
			i = &Icon{
				Data:      mpkg.Icon.Data,
				MediaType: mpkg.Icon.MediaType,
			}
		}
		defaultChannel := ""
		if mpkg.DefaultChannel != nil {
			defaultChannel = mpkg.DefaultChannel.Name
		}
		cfg.Packages = append(cfg.Packages, Package{
			Schema:         schemaPackage,
			Name:           mpkg.Name,
			DefaultChannel: defaultChannel,
			Icon:           i,
			Description:    mpkg.Description,
		})
		cfg.Bundles = append(cfg.Bundles, bundles...)
	}

	sort.Slice(cfg.Packages, func(i, j int) bool {
		return cfg.Packages[i].Name < cfg.Packages[j].Name
	})
	sort.Slice(cfg.Bundles, func(i, j int) bool {
		return cfg.Bundles[i].Name < cfg.Bundles[j].Name
	})

	return cfg
}

func traverseModelChannels(mpkg model.Package) []Bundle {
	bundles := map[string]*Bundle{}

	for _, ch := range mpkg.Channels {
		for _, chb := range ch.Bundles {
			b, ok := bundles[chb.Name]
			if !ok {
				b = &Bundle{
					Schema:        schemaBundle,
					Name:          chb.Name,
					Package:       chb.Package.Name,
					Image:         chb.Image,
					RelatedImages: modelRelatedImagesToRelatedImages(chb.RelatedImages),
					CsvJSON:       chb.CsvJSON,
					Objects:       chb.Objects,
				}
				bundles[b.Name] = b
			}
			b.Properties = append(b.Properties, chb.Properties...)
		}
	}

	var out []Bundle
	for _, b := range bundles {
		b.Properties = property.Deduplicate(b.Properties)
		out = append(out, *b)
	}
	return out
}

func modelRelatedImagesToRelatedImages(relatedImages []model.RelatedImage) []RelatedImage {
	var out []RelatedImage
	for _, ri := range relatedImages {
		out = append(out, RelatedImage{
			Name:  ri.Name,
			Image: ri.Image,
		})
	}
	return out
}
