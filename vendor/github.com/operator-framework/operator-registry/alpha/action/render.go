package action

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/h2non/filetype"
	"github.com/h2non/filetype/matchers"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/operator-framework/operator-registry/alpha/declcfg"
	"github.com/operator-framework/operator-registry/alpha/property"
	"github.com/operator-framework/operator-registry/pkg/containertools"
	"github.com/operator-framework/operator-registry/pkg/image"
	"github.com/operator-framework/operator-registry/pkg/image/containerdregistry"
	"github.com/operator-framework/operator-registry/pkg/lib/bundle"
	"github.com/operator-framework/operator-registry/pkg/registry"
	"github.com/operator-framework/operator-registry/pkg/sqlite"
)

var logDeprecationMessage sync.Once

type RefType uint

const (
	RefBundleImage RefType = 1 << iota
	RefSqliteImage
	RefSqliteFile
	RefDCImage
	RefDCDir

	RefAll = 0
)

func (r RefType) Allowed(refType RefType) bool {
	return r == RefAll || r&refType == refType
}

var ErrNotAllowed = errors.New("not allowed")

type Render struct {
	Refs           []string
	Registry       image.Registry
	AllowedRefMask RefType

	skipSqliteDeprecationLog bool
}

func nullLogger() *logrus.Entry {
	logger := logrus.New()
	logger.SetOutput(ioutil.Discard)
	return logrus.NewEntry(logger)
}

func (r Render) Run(ctx context.Context) (*declcfg.DeclarativeConfig, error) {
	if r.skipSqliteDeprecationLog {
		// exhaust once with a no-op function.
		logDeprecationMessage.Do(func() {})
	}
	if r.Registry == nil {
		reg, err := r.createRegistry()
		if err != nil {
			return nil, fmt.Errorf("create registry: %v", err)
		}
		defer reg.Destroy()
		r.Registry = reg
	}

	var cfgs []declcfg.DeclarativeConfig
	for _, ref := range r.Refs {
		cfg, err := r.renderReference(ctx, ref)
		if err != nil {
			return nil, fmt.Errorf("render reference %q: %w", ref, err)
		}
		renderBundleObjects(cfg)

		for _, b := range cfg.Bundles {
			sort.Slice(b.RelatedImages, func(i, j int) bool {
				return b.RelatedImages[i].Image < b.RelatedImages[j].Image
			})
		}

		cfgs = append(cfgs, *cfg)
	}

	return combineConfigs(cfgs), nil
}

func (r Render) createRegistry() (*containerdregistry.Registry, error) {
	cacheDir, err := os.MkdirTemp("", "render-registry-")
	if err != nil {
		return nil, fmt.Errorf("create tempdir: %v", err)
	}

	reg, err := containerdregistry.NewRegistry(
		containerdregistry.WithCacheDir(cacheDir),

		// The containerd registry impl is somewhat verbose, even on the happy path,
		// so discard all logger logs. Any important failures will be returned from
		// registry methods and eventually logged as fatal errors.
		containerdregistry.WithLog(nullLogger()),
	)
	if err != nil {
		return nil, err
	}
	return reg, nil
}

func (r Render) renderReference(ctx context.Context, ref string) (*declcfg.DeclarativeConfig, error) {
	if stat, serr := os.Stat(ref); serr == nil {
		if stat.IsDir() {
			if !r.AllowedRefMask.Allowed(RefDCDir) {
				return nil, fmt.Errorf("cannot render declarative config directory: %w", ErrNotAllowed)
			}
			return declcfg.LoadFS(os.DirFS(ref))
		} else {
			// The only supported file type is an sqlite DB file,
			// since declarative configs will be in a directory.
			if err := checkDBFile(ref); err != nil {
				return nil, err
			}
			if !r.AllowedRefMask.Allowed(RefSqliteFile) {
				return nil, fmt.Errorf("cannot render sqlite file: %w", ErrNotAllowed)
			}
			return sqliteToDeclcfg(ctx, ref)
		}
	}
	return r.imageToDeclcfg(ctx, ref)
}

func (r Render) imageToDeclcfg(ctx context.Context, imageRef string) (*declcfg.DeclarativeConfig, error) {
	ref := image.SimpleReference(imageRef)
	if err := r.Registry.Pull(ctx, ref); err != nil {
		return nil, err
	}
	labels, err := r.Registry.Labels(ctx, ref)
	if err != nil {
		return nil, err
	}
	tmpDir, err := ioutil.TempDir("", "render-unpack-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)
	if err := r.Registry.Unpack(ctx, ref, tmpDir); err != nil {
		return nil, err
	}

	var cfg *declcfg.DeclarativeConfig
	if dbFile, ok := labels[containertools.DbLocationLabel]; ok {
		if !r.AllowedRefMask.Allowed(RefSqliteImage) {
			return nil, fmt.Errorf("cannot render sqlite image: %w", ErrNotAllowed)
		}
		cfg, err = sqliteToDeclcfg(ctx, filepath.Join(tmpDir, dbFile))
		if err != nil {
			return nil, err
		}
	} else if configsDir, ok := labels[containertools.ConfigsLocationLabel]; ok {
		if !r.AllowedRefMask.Allowed(RefDCImage) {
			return nil, fmt.Errorf("cannot render declarative config image: %w", ErrNotAllowed)
		}
		cfg, err = declcfg.LoadFS(os.DirFS(filepath.Join(tmpDir, configsDir)))
		if err != nil {
			return nil, err
		}
	} else if _, ok := labels[bundle.PackageLabel]; ok {
		if !r.AllowedRefMask.Allowed(RefBundleImage) {
			return nil, fmt.Errorf("cannot render bundle image: %w", ErrNotAllowed)
		}
		img, err := registry.NewImageInput(ref, tmpDir)
		if err != nil {
			return nil, err
		}

		cfg, err = bundleToDeclcfg(img.Bundle)
		if err != nil {
			return nil, err
		}
	} else {
		labelKeys := sets.StringKeySet(labels)
		labelVals := []string{}
		for _, k := range labelKeys.List() {
			labelVals = append(labelVals, fmt.Sprintf("  %s=%s", k, labels[k]))
		}
		if len(labelVals) > 0 {
			return nil, fmt.Errorf("render %q: image type could not be determined, found labels\n%s", ref, strings.Join(labelVals, "\n"))
		} else {
			return nil, fmt.Errorf("render %q: image type could not be determined: image has no labels", ref)
		}
	}
	return cfg, nil
}

// checkDBFile returns an error if ref is not an sqlite3 database.
func checkDBFile(ref string) error {
	typ, err := filetype.MatchFile(ref)
	if err != nil {
		return err
	}
	if typ != matchers.TypeSqlite {
		return fmt.Errorf("ref %q has unsupported file type: %s", ref, typ)
	}
	return nil
}

func sqliteToDeclcfg(ctx context.Context, dbFile string) (*declcfg.DeclarativeConfig, error) {
	logDeprecationMessage.Do(func() {
		sqlite.LogSqliteDeprecation()
	})

	db, err := sqlite.Open(dbFile)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	migrator, err := sqlite.NewSQLLiteMigrator(db)
	if err != nil {
		return nil, err
	}
	if migrator == nil {
		return nil, fmt.Errorf("failed to load migrator")
	}

	if err := migrator.Migrate(ctx); err != nil {
		return nil, err
	}

	q := sqlite.NewSQLLiteQuerierFromDb(db)
	m, err := sqlite.ToModel(ctx, q)
	if err != nil {
		return nil, err
	}

	cfg := declcfg.ConvertFromModel(m)

	if err := populateDBRelatedImages(ctx, &cfg, db); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func populateDBRelatedImages(ctx context.Context, cfg *declcfg.DeclarativeConfig, db *sql.DB) error {
	rows, err := db.QueryContext(ctx, "SELECT image, operatorbundle_name FROM related_image")
	if err != nil {
		return err
	}
	defer rows.Close()

	images := map[string]sets.String{}
	for rows.Next() {
		var (
			img        sql.NullString
			bundleName sql.NullString
		)
		if err := rows.Scan(&img, &bundleName); err != nil {
			return err
		}
		if !img.Valid || !bundleName.Valid {
			continue
		}
		m, ok := images[bundleName.String]
		if !ok {
			m = sets.NewString()
		}
		m.Insert(img.String)
		images[bundleName.String] = m
	}

	for i, b := range cfg.Bundles {
		ris, ok := images[b.Name]
		if !ok {
			continue
		}
		for _, ri := range b.RelatedImages {
			if ris.Has(ri.Image) {
				ris.Delete(ri.Image)
			}
		}
		for ri := range ris {
			cfg.Bundles[i].RelatedImages = append(cfg.Bundles[i].RelatedImages, declcfg.RelatedImage{Image: ri})
		}
	}
	return nil
}

func bundleToDeclcfg(bundle *registry.Bundle) (*declcfg.DeclarativeConfig, error) {
	objs, props, err := registry.ObjectsAndPropertiesFromBundle(bundle)
	if err != nil {
		return nil, fmt.Errorf("get properties for bundle %q: %v", bundle.Name, err)
	}
	relatedImages, err := getRelatedImages(bundle)
	if err != nil {
		return nil, fmt.Errorf("get related images for bundle %q: %v", bundle.Name, err)
	}
	var csvJson []byte
	for _, obj := range bundle.Objects {
		if obj.GetKind() == "ClusterServiceVersion" {
			csvJson, err = json.Marshal(obj)
			if err != nil {
				return nil, fmt.Errorf("marshal CSV JSON for bundle %q: %v", bundle.Name, err)
			}
		}
	}

	dBundle := declcfg.Bundle{
		Schema:        "olm.bundle",
		Name:          bundle.Name,
		Package:       bundle.Package,
		Image:         bundle.BundleImage,
		Properties:    props,
		RelatedImages: relatedImages,
		Objects:       objs,
		CsvJSON:       string(csvJson),
	}

	return &declcfg.DeclarativeConfig{Bundles: []declcfg.Bundle{dBundle}}, nil
}

func getRelatedImages(b *registry.Bundle) ([]declcfg.RelatedImage, error) {
	csv, err := b.ClusterServiceVersion()
	if err != nil {
		return nil, err
	}

	var objmap map[string]*json.RawMessage
	if err = json.Unmarshal(csv.Spec, &objmap); err != nil {
		return nil, err
	}

	var relatedImages []declcfg.RelatedImage
	rawValue, ok := objmap["relatedImages"]
	if ok && rawValue != nil {
		if err = json.Unmarshal(*rawValue, &relatedImages); err != nil {
			return nil, err
		}
	}

	// Keep track of the images we've already found, so that we don't add
	// them multiple times.
	allImages := sets.NewString()
	for _, ri := range relatedImages {
		allImages = allImages.Insert(ri.Image)
	}

	if !allImages.Has(b.BundleImage) {
		relatedImages = append(relatedImages, declcfg.RelatedImage{
			Image: b.BundleImage,
		})
	}

	opImages, err := csv.GetOperatorImages()
	if err != nil {
		return nil, err
	}
	for img := range opImages {
		if !allImages.Has(img) {
			relatedImages = append(relatedImages, declcfg.RelatedImage{
				Image: img,
			})
		}
		allImages = allImages.Insert(img)
	}

	return relatedImages, nil
}

func renderBundleObjects(cfg *declcfg.DeclarativeConfig) {
	for bi, b := range cfg.Bundles {
		props := b.Properties[:0]
		for _, p := range b.Properties {
			if p.Type != property.TypeBundleObject {
				props = append(props, p)
			}
		}

		for _, obj := range b.Objects {
			props = append(props, property.MustBuildBundleObjectData([]byte(obj)))
		}
		cfg.Bundles[bi].Properties = props
	}
}

func combineConfigs(cfgs []declcfg.DeclarativeConfig) *declcfg.DeclarativeConfig {
	out := &declcfg.DeclarativeConfig{}
	for _, in := range cfgs {
		out.Packages = append(out.Packages, in.Packages...)
		out.Channels = append(out.Channels, in.Channels...)
		out.Bundles = append(out.Bundles, in.Bundles...)
		out.Others = append(out.Others, in.Others...)
	}
	return out
}
