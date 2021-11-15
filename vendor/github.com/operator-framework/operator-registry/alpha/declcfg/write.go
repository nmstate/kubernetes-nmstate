package declcfg

import (
	"bytes"
	"encoding/json"
	"io"
	"sort"

	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/yaml"
)

func WriteJSON(cfg DeclarativeConfig, w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "    ")
	enc.SetEscapeHTML(false)
	return writeToEncoder(cfg, enc)
}

func WriteYAML(cfg DeclarativeConfig, w io.Writer) error {
	enc := newYAMLEncoder(w)
	enc.SetEscapeHTML(false)
	return writeToEncoder(cfg, enc)
}

type yamlEncoder struct {
	w          io.Writer
	escapeHTML bool
}

func newYAMLEncoder(w io.Writer) *yamlEncoder {
	return &yamlEncoder{w, true}
}

func (e *yamlEncoder) SetEscapeHTML(on bool) {
	e.escapeHTML = on
}

func (e *yamlEncoder) Encode(v interface{}) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(e.escapeHTML)
	if err := enc.Encode(v); err != nil {
		return err
	}
	yamlData, err := yaml.JSONToYAML(buf.Bytes())
	if err != nil {
		return err
	}
	yamlData = append([]byte("---\n"), yamlData...)
	_, err = e.w.Write(yamlData)
	return err
}

type encoder interface {
	Encode(interface{}) error
}

func writeToEncoder(cfg DeclarativeConfig, enc encoder) error {
	pkgNames := sets.NewString()

	packagesByName := map[string][]Package{}
	for _, p := range cfg.Packages {
		pkgName := p.Name
		pkgNames.Insert(pkgName)
		packagesByName[pkgName] = append(packagesByName[pkgName], p)
	}
	channelsByPackage := map[string][]Channel{}
	for _, c := range cfg.Channels {
		pkgName := c.Package
		pkgNames.Insert(pkgName)
		channelsByPackage[pkgName] = append(channelsByPackage[pkgName], c)
	}
	bundlesByPackage := map[string][]Bundle{}
	for _, b := range cfg.Bundles {
		pkgName := b.Package
		pkgNames.Insert(pkgName)
		bundlesByPackage[pkgName] = append(bundlesByPackage[pkgName], b)
	}
	othersByPackage := map[string][]Meta{}
	for _, o := range cfg.Others {
		pkgName := o.Package
		pkgNames.Insert(pkgName)
		othersByPackage[pkgName] = append(othersByPackage[pkgName], o)
	}

	for _, pName := range pkgNames.List() {
		if len(pName) == 0 {
			continue
		}
		pkgs := packagesByName[pName]
		for _, p := range pkgs {
			if err := enc.Encode(p); err != nil {
				return err
			}
		}

		channels := channelsByPackage[pName]
		sort.Slice(channels, func(i, j int) bool {
			return channels[i].Name < channels[j].Name
		})
		for _, c := range channels {
			if err := enc.Encode(c); err != nil {
				return err
			}
		}

		bundles := bundlesByPackage[pName]
		sort.Slice(bundles, func(i, j int) bool {
			return bundles[i].Name < bundles[j].Name
		})
		for _, b := range bundles {
			if err := enc.Encode(b); err != nil {
				return err
			}
		}

		others := othersByPackage[pName]
		sort.SliceStable(others, func(i, j int) bool {
			return others[i].Schema < others[j].Schema
		})
		for _, o := range others {
			if err := enc.Encode(o); err != nil {
				return err
			}
		}
	}

	for _, o := range othersByPackage[""] {
		if err := enc.Encode(o); err != nil {
			return err
		}
	}
	return nil
}
