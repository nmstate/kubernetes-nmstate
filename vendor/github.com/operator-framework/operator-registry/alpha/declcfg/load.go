package declcfg

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/joelanford/ignore"
	"github.com/operator-framework/api/pkg/operators"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/operator-framework/operator-registry/alpha/property"
)

const (
	indexIgnoreFilename = ".indexignore"
)

type WalkFunc func(path string, cfg *DeclarativeConfig, err error) error

// WalkFS walks root using a gitignore-style filename matcher to skip files
// that match patterns found in .indexignore files found throughout the filesystem.
// It calls walkFn for each declarative config file it finds. If WalkFS encounters
// an error loading or parsing any file, the error will be immediately returned.
func WalkFS(root fs.FS, walkFn WalkFunc) error {
	if root == nil {
		return fmt.Errorf("no declarative config filesystem provided")
	}
	matcher, err := ignore.NewMatcher(root, indexIgnoreFilename)
	if err != nil {
		return err
	}

	return fs.WalkDir(root, ".", func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return walkFn(path, nil, err)
		}
		// avoid validating a directory, an .indexignore file, or any file that matches
		// an ignore pattern outlined in a .indexignore file.
		if info.IsDir() || info.Name() == indexIgnoreFilename || matcher.Match(path, false) {
			return nil
		}
		file, err := root.Open(path)
		if err != nil {
			return walkFn(path, nil, err)
		}
		defer file.Close()
		cfg, err := readYAMLOrJSON(file)
		if err != nil {
			return walkFn(path, cfg, err)
		}
		if err := readBundleObjects(cfg.Bundles, root, path); err != nil {
			return fmt.Errorf("read bundle objects: %v", err)
		}
		return walkFn(path, cfg, err)
	})
}

// LoadFS loads a declarative config from the provided root FS. LoadFS walks the
// filesystem from root and uses a gitignore-style filename matcher to skip files
// that match patterns found in .indexignore files found throughout the filesystem.
// If LoadFS encounters an error loading or parsing any file, the error will be
// immediately returned.
func LoadFS(root fs.FS) (*DeclarativeConfig, error) {
	cfg := &DeclarativeConfig{}
	if err := WalkFS(root, func(path string, fcfg *DeclarativeConfig, err error) error {
		if err != nil {
			return err
		}
		cfg.Packages = append(cfg.Packages, fcfg.Packages...)
		cfg.Channels = append(cfg.Channels, fcfg.Channels...)
		cfg.Bundles = append(cfg.Bundles, fcfg.Bundles...)
		cfg.Others = append(cfg.Others, fcfg.Others...)
		return nil
	}); err != nil {
		return nil, err
	}
	return cfg, nil
}

func readBundleObjects(bundles []Bundle, root fs.FS, path string) error {
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

func readYAMLOrJSON(r io.Reader) (*DeclarativeConfig, error) {
	cfg := &DeclarativeConfig{}
	dec := yaml.NewYAMLOrJSONDecoder(r, 4096)
	for {
		doc := json.RawMessage{}
		if err := dec.Decode(&doc); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		doc = []byte(strings.NewReplacer(`\u003c`, "<", `\u003e`, ">", `\u0026`, "&").Replace(string(doc)))

		var in Meta
		if err := json.Unmarshal(doc, &in); err != nil {
			return nil, err
		}

		switch in.Schema {
		case schemaPackage:
			var p Package
			if err := json.Unmarshal(doc, &p); err != nil {
				return nil, fmt.Errorf("parse package: %v", err)
			}
			cfg.Packages = append(cfg.Packages, p)
		case schemaChannel:
			var c Channel
			if err := json.Unmarshal(doc, &c); err != nil {
				return nil, fmt.Errorf("parse channel: %v", err)
			}
			cfg.Channels = append(cfg.Channels, c)
		case schemaBundle:
			var b Bundle
			if err := json.Unmarshal(doc, &b); err != nil {
				return nil, fmt.Errorf("parse bundle: %v", err)
			}
			cfg.Bundles = append(cfg.Bundles, b)
		case "":
			return nil, fmt.Errorf("object '%s' is missing root schema field", string(doc))
		default:
			cfg.Others = append(cfg.Others, in)
		}
	}
	return cfg, nil
}
