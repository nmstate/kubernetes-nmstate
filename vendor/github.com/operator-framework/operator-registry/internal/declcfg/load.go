package declcfg

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ghodss/yaml"
	"github.com/operator-framework/api/pkg/operators"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/operator-framework/operator-registry/internal/property"
)

func LoadDir(configDir string) (*DeclarativeConfig, error) {
	w := &dirWalker{}
	return loadFS(configDir, w)
}

func loadFS(root string, w fsWalker) (*DeclarativeConfig, error) {
	cfg := &DeclarativeConfig{}
	if err := w.WalkFiles(root, func(path string, r io.Reader) error {
		fileCfg, err := readJSON(r)
		if err != nil {
			return fmt.Errorf("could not load config file %q: %v", path, err)
		}
		if err := readBundleObjects(fileCfg.Bundles, root, path); err != nil {
			return fmt.Errorf("read bundle objects: %v", err)
		}
		cfg.Packages = append(cfg.Packages, fileCfg.Packages...)
		cfg.Bundles = append(cfg.Bundles, fileCfg.Bundles...)
		cfg.Others = append(cfg.Others, fileCfg.Others...)

		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to read declarative configs dir: %v", err)
	}
	return cfg, nil
}

func readBundleObjects(bundles []Bundle, root, path string) error {
	for bi, b := range bundles {
		props, err := property.Parse(b.Properties)
		if err != nil {
			return fmt.Errorf("parse properties for bundle %q: %v", b.Name, err)
		}
		for oi, obj := range props.BundleObjects {
			d, err := obj.GetData(root, filepath.Dir(path))
			if err != nil {
				return fmt.Errorf("get data for bundle object[%d]: %v", oi, err)
			}
			bundles[bi].Objects = append(bundles[bi].Objects, string(d))
		}
		bundles[bi].CsvJSON = extractCSV(bundles[bi].Objects)
	}
	return nil
}

func extractCSV(objs []string) string {
	for _, obj := range objs {
		u := unstructured.Unstructured{}
		if err := yaml.Unmarshal([]byte(obj), &u); err != nil {
			continue
		}
		if u.GetKind() == operators.ClusterServiceVersionKind {
			return obj
		}
	}
	return ""
}

func readJSON(r io.Reader) (*DeclarativeConfig, error) {
	cfg := &DeclarativeConfig{}
	dec := json.NewDecoder(r)
	for dec.More() {
		doc := &json.RawMessage{}
		if err := dec.Decode(doc); err != nil {
			return cfg, nil
		}

		var in Meta
		if err := json.Unmarshal(*doc, &in); err != nil {
			// Ignore JSON blobs if they are not parsable as meta objects.
			continue
		}

		switch in.Schema {
		case schemaPackage:
			var p Package
			if err := json.Unmarshal(*doc, &p); err != nil {
				return nil, fmt.Errorf("parse package at offset %d: %v", dec.InputOffset(), err)
			}
			cfg.Packages = append(cfg.Packages, p)
		case schemaBundle:
			var b Bundle
			if err := json.Unmarshal(*doc, &b); err != nil {
				return nil, fmt.Errorf("parse bundle at offset %d: %v", dec.InputOffset(), err)
			}
			cfg.Bundles = append(cfg.Bundles, b)
		case "":
			// Ignore meta blobs that don't have a schema.
			continue
		default:
			cfg.Others = append(cfg.Others, in)
		}
	}
	return cfg, nil
}

type fsWalker interface {
	WalkFiles(root string, f func(path string, r io.Reader) error) error
}

type dirWalker struct{}

func (w dirWalker) WalkFiles(root string, f func(string, io.Reader) error) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		return f(path, file)
	})
}
