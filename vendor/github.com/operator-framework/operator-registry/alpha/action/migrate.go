package action

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/operator-framework/operator-registry/alpha/declcfg"
	"github.com/operator-framework/operator-registry/pkg/image"
)

type Migrate struct {
	CatalogRef string
	OutputDir  string

	WriteFunc WriteFunc
	FileExt   string
	Registry  image.Registry
}

type WriteFunc func(config declcfg.DeclarativeConfig, w io.Writer) error

func (m Migrate) Run(ctx context.Context) error {
	entries, err := ioutil.ReadDir(m.OutputDir)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if len(entries) > 0 {
		return fmt.Errorf("output dir %q must be empty", m.OutputDir)
	}

	r := Render{
		Refs: []string{m.CatalogRef},

		// Only allow sqlite images and files to be migrated. Other types cannot
		// always be migrated cleanly because they may contain file references.
		// Rendered sqlite databases never contain file references.
		AllowedRefMask: RefSqliteImage | RefSqliteFile,

		skipSqliteDeprecationLog: true,
	}
	if m.Registry != nil {
		r.Registry = m.Registry
	}

	cfg, err := r.Run(ctx)
	if err != nil {
		return fmt.Errorf("render catalog image: %w", err)
	}

	return writeToFS(*cfg, m.OutputDir, m.WriteFunc, m.FileExt)
}

func writeToFS(cfg declcfg.DeclarativeConfig, rootDir string, writeFunc WriteFunc, fileExt string) error {
	channelsByPackage := map[string][]declcfg.Channel{}
	for _, c := range cfg.Channels {
		channelsByPackage[c.Package] = append(channelsByPackage[c.Package], c)
	}
	bundlesByPackage := map[string][]declcfg.Bundle{}
	for _, b := range cfg.Bundles {
		bundlesByPackage[b.Package] = append(bundlesByPackage[b.Package], b)
	}

	if err := os.MkdirAll(rootDir, 0777); err != nil {
		return err
	}

	for _, p := range cfg.Packages {
		fcfg := declcfg.DeclarativeConfig{
			Packages: []declcfg.Package{p},
			Channels: channelsByPackage[p.Name],
			Bundles:  bundlesByPackage[p.Name],
		}
		pkgDir := filepath.Join(rootDir, p.Name)
		if err := os.MkdirAll(pkgDir, 0777); err != nil {
			return err
		}
		filename := filepath.Join(pkgDir, fmt.Sprintf("catalog%s", fileExt))
		if err := writeFile(fcfg, filename, writeFunc); err != nil {
			return err
		}
	}
	return nil
}

func writeFile(cfg declcfg.DeclarativeConfig, filename string, writeFunc WriteFunc) error {
	buf := &bytes.Buffer{}
	if err := writeFunc(cfg, buf); err != nil {
		return fmt.Errorf("write to buffer for %q: %v", filename, err)
	}
	if err := ioutil.WriteFile(filename, buf.Bytes(), 0666); err != nil {
		return fmt.Errorf("write file %q: %v", filename, err)
	}
	return nil
}
