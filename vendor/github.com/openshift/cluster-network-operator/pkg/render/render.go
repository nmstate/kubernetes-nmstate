package render

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/pkg/errors"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

type RenderData struct {
	Funcs template.FuncMap
	Data  map[string]interface{}
}

func MakeRenderData() RenderData {
	return RenderData{
		Funcs: template.FuncMap{},
		Data:  map[string]interface{}{},
	}
}

// RenderDir will render all manifests in a directory, descending in to subdirectories
// It will perform template substitutions based on the data supplied by the RenderData
func RenderDir(manifestDir string, d *RenderData) ([]*unstructured.Unstructured, error) {
	return RenderDirs([]string{manifestDir}, d)
}

// RenderDirs renders multiple directories, but sorts the discovered files *globally* first.
// In other words, if you have the structure
// - a/001.yaml
// - a/003.yaml
// - b/002.yaml
// It will still render 001, 002, and 003 in order.
func RenderDirs(manifestDirs []string, d *RenderData) ([]*unstructured.Unstructured, error) {
	out := []*unstructured.Unstructured{}

	files := byFilename{}
	for _, dir := range manifestDirs {
		if err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}

			// Skip non-manifest files
			if !(strings.HasSuffix(path, ".yml") || strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".json")) {
				return nil
			}

			files = append(files, path)

			return nil
		}); err != nil {
			return nil, errors.Wrap(err, "error listing manifests")
		}
	}
	// sort files by filename, not full path
	sort.Sort(files)

	for _, path := range files {
		objs, err := RenderTemplate(path, d)
		if err != nil {
			return nil, fmt.Errorf("failed to render file %s: %w", path, err)
		}
		out = append(out, objs...)
	}

	return out, nil
}

// sorting boilerplate

type byFilename []string

func (a byFilename) Len() int      { return len(a) }
func (a byFilename) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

// sort by filename w/o dir
func (a byFilename) Less(i, j int) bool {
	_, p1 := filepath.Split(a[i])
	_, p2 := filepath.Split(a[j])
	return p1 < p2
}

// RenderTemplate reads, renders, and attempts to parse a yaml or
// json file representing one or more k8s api objects
func RenderTemplate(path string, d *RenderData) ([]*unstructured.Unstructured, error) {
	tmpl := template.New(path).Option("missingkey=error")
	if d.Funcs != nil {
		tmpl.Funcs(d.Funcs)
	}

	// Add universal functions
	tmpl.Funcs(template.FuncMap{"getOr": getOr, "isSet": isSet, "iniEscapeCharacters": iniEscapeCharacters})
	tmpl.Funcs(sprig.TxtFuncMap())

	source, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read manifest %s", path)
	}

	if _, err := tmpl.Parse(string(source)); err != nil {
		return nil, errors.Wrapf(err, "failed to parse manifest %s as template", path)
	}

	rendered := bytes.Buffer{}
	if err := tmpl.Execute(&rendered, d.Data); err != nil {
		return nil, errors.Wrapf(err, "failed to render manifest %s", path)
	}

	out := []*unstructured.Unstructured{}

	// special case - if the entire file is whitespace, skip
	if len(strings.TrimSpace(rendered.String())) == 0 {
		return out, nil
	}

	decoder := yaml.NewYAMLOrJSONDecoder(&rendered, 4096)
	for {
		u := unstructured.Unstructured{}
		if err := decoder.Decode(&u); err != nil {
			if err == io.EOF {
				break
			}
			return nil, errors.Wrapf(err, "failed to unmarshal manifest %s", path)
		}
		out = append(out, &u)
	}

	return out, nil
}
