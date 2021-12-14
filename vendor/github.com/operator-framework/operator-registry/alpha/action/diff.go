package action

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/blang/semver/v4"
	"github.com/sirupsen/logrus"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/operator-framework/operator-registry/alpha/declcfg"
	"github.com/operator-framework/operator-registry/alpha/model"
	"github.com/operator-framework/operator-registry/pkg/image"
)

type Diff struct {
	Registry image.Registry

	OldRefs []string
	NewRefs []string
	// SkipDependencies directs Run() to not include dependencies
	// of bundles included in the diff if true.
	SkipDependencies bool

	IncludeConfig DiffIncludeConfig
	// IncludeAdditively catalog objects specified in IncludeConfig.
	IncludeAdditively bool

	Logger *logrus.Entry
}

func (a Diff) Run(ctx context.Context) (*declcfg.DeclarativeConfig, error) {
	if err := a.validate(); err != nil {
		return nil, err
	}

	// Disallow bundle refs.
	mask := RefDCDir | RefDCImage | RefSqliteFile | RefSqliteImage

	// Heads-only mode does not require an old ref, so there may be nothing to render.
	var oldModel model.Model
	if len(a.OldRefs) != 0 {
		oldRender := Render{Refs: a.OldRefs, Registry: a.Registry, AllowedRefMask: mask}
		oldCfg, err := oldRender.Run(ctx)
		if err != nil {
			if errors.Is(err, ErrNotAllowed) {
				return nil, fmt.Errorf("%w (diff does not permit direct bundle references)", err)
			}
			return nil, fmt.Errorf("error rendering old refs: %v", err)
		}
		oldModel, err = declcfg.ConvertToModel(*oldCfg)
		if err != nil {
			return nil, fmt.Errorf("error converting old declarative config to model: %v", err)
		}
	}

	newRender := Render{Refs: a.NewRefs, Registry: a.Registry, AllowedRefMask: mask}
	newCfg, err := newRender.Run(ctx)
	if err != nil {
		if errors.Is(err, ErrNotAllowed) {
			return nil, fmt.Errorf("%w (diff does not permit direct bundle references)", err)
		}
		return nil, fmt.Errorf("error rendering new refs: %v", err)
	}
	newModel, err := declcfg.ConvertToModel(*newCfg)
	if err != nil {
		return nil, fmt.Errorf("error converting new declarative config to model: %v", err)
	}

	g := &declcfg.DiffGenerator{
		Logger:            a.Logger,
		SkipDependencies:  a.SkipDependencies,
		Includer:          convertIncludeConfigToIncluder(a.IncludeConfig),
		IncludeAdditively: a.IncludeAdditively,
	}
	diffModel, err := g.Run(oldModel, newModel)
	if err != nil {
		return nil, fmt.Errorf("error generating diff: %v", err)
	}

	cfg := declcfg.ConvertFromModel(diffModel)
	return &cfg, nil
}

func (p Diff) validate() error {
	if len(p.NewRefs) == 0 {
		return fmt.Errorf("no new refs to diff")
	}
	return nil
}

// DiffIncludeConfig configures Diff.Run() to include a set of packages,
// channels, and/or bundles/versions in the output DeclarativeConfig.
// These override other diff mechanisms. For example, if running in
// heads-only mode but package "foo" channel "stable" is specified,
// the entire "stable" channel (all channel bundles) is added to the output.
type DiffIncludeConfig struct {
	// Packages to include.
	Packages []DiffIncludePackage `json:"packages" yaml:"packages"`
}

// DiffIncludePackage contains a name (required) and channels and/or versions
// (optional) to include in the diff. The full package is only included if no channels
// or versions are specified.
type DiffIncludePackage struct {
	// Name of package.
	Name string `json:"name" yaml:"name"`
	// Channels to include.
	Channels []DiffIncludeChannel `json:"channels,omitempty" yaml:"channels,omitempty"`
	// Versions to include. All channels containing these versions
	// are parsed for an upgrade graph.
	Versions []semver.Version `json:"versions,omitempty" yaml:"versions,omitempty"`
	// Bundles are bundle names to include. All channels containing these bundles
	// are parsed for an upgrade graph.
	// Set this field only if the named bundle has no semantic version metadata.
	Bundles []string `json:"bundles,omitempty" yaml:"bundles,omitempty"`
}

// DiffIncludeChannel contains a name (required) and versions (optional)
// to include in the diff. The full channel is only included if no versions are specified.
type DiffIncludeChannel struct {
	// Name of channel.
	Name string `json:"name" yaml:"name"`
	// Versions to include.
	Versions []semver.Version `json:"versions,omitempty" yaml:"versions,omitempty"`
	// Bundles are bundle names to include.
	// Set this field only if the named bundle has no semantic version metadata.
	Bundles []string `json:"bundles,omitempty" yaml:"bundles,omitempty"`
}

// LoadDiffIncludeConfig loads a (YAML or JSON) DiffIncludeConfig from r.
func LoadDiffIncludeConfig(r io.Reader) (c DiffIncludeConfig, err error) {
	dec := yaml.NewYAMLOrJSONDecoder(r, 8)
	if err := dec.Decode(&c); err != nil {
		return DiffIncludeConfig{}, err
	}

	if len(c.Packages) == 0 {
		return c, fmt.Errorf("must specify at least one package in include config")
	}

	var errs []error
	for pkgI, pkg := range c.Packages {
		if pkg.Name == "" {
			errs = append(errs, fmt.Errorf("package at index %v requires a name", pkgI))
			continue
		}
		for chI, ch := range pkg.Channels {
			if ch.Name == "" {
				errs = append(errs, fmt.Errorf("package %s: channel at index %v requires a name", pkg.Name, chI))
				continue
			}
		}
	}
	return c, utilerrors.NewAggregate(errs)
}

func convertIncludeConfigToIncluder(c DiffIncludeConfig) (includer declcfg.DiffIncluder) {
	includer.Packages = make([]declcfg.DiffIncludePackage, len(c.Packages))
	for pkgI, cpkg := range c.Packages {
		pkg := &includer.Packages[pkgI]
		pkg.Name = cpkg.Name
		pkg.AllChannels.Versions = cpkg.Versions
		pkg.AllChannels.Bundles = cpkg.Bundles

		if len(cpkg.Channels) != 0 {
			pkg.Channels = make([]declcfg.DiffIncludeChannel, len(cpkg.Channels))
			for chI, cch := range cpkg.Channels {
				ch := &pkg.Channels[chI]
				ch.Name = cch.Name
				ch.Versions = cch.Versions
				ch.Bundles = cch.Bundles
			}
		}
	}
	return includer
}
