package appregistry

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/operator-framework/operator-registry/pkg/apprclient"
)

type AppregistryBuildOptions struct {
	Appender ImageAppender

	From, To            string
	AuthToken           string
	AppRegistryEndpoint string
	AppRegistryOrg      string
	DatabasePath        string
	CacheDir            string

	// derived
	CleanOutput bool
	ManifestDir string
	DatabaseDir string

	Client apprclient.Client
}

func (o *AppregistryBuildOptions) Validate() error {
	// TODO: better validation

	if o.AppRegistryEndpoint == "" {
		return fmt.Errorf("app-registry must be a valid app-registry endpoint")
	}
	if o.AppRegistryOrg == "" {
		return fmt.Errorf("app-registry org (namespace) must be specified")
	}
	if o.Appender == nil {
		return fmt.Errorf("appregistry image builder can't run without an appender")
	}
	if o.From == "" {
		return fmt.Errorf("base image required (--from)")
	}
	if o.DatabasePath == "" {
		return fmt.Errorf("database must have a path to save to")
	}
	if o.ManifestDir == "" {
		return fmt.Errorf("local manifest directory required")
	}
	if o.DatabaseDir == "" {
		return fmt.Errorf("local database directory required")
	}
	if o.Client == nil {
		return fmt.Errorf("app-registry client must not be nil")
	}

	return nil
}

func (o *AppregistryBuildOptions) Complete() error {
	// if a user hasn't specified a specific cache directory, generate a temporary one
	if o.CacheDir == "" {
		tmp, err := ioutil.TempDir("", "cache-")
		if err != nil {
			return err
		}
		o.CacheDir = tmp

		// clean up temporary directories
		o.CleanOutput = true
	} else if err := os.MkdirAll(o.CacheDir, os.ModePerm); err != nil && !os.IsExist(err) {
		return err
	}

	// build a separate path for manifests and the built database, so that building is idempotent
	if o.ManifestDir == "" {
		manifestDir, err := ioutil.TempDir(o.CacheDir, "manifests-")
		if err != nil {
			return err
		}
		o.ManifestDir = manifestDir
	}

	if o.DatabaseDir == "" {
		databaseDir, err := ioutil.TempDir("", "db-")
		if err != nil {
			return err
		}
		o.DatabaseDir = databaseDir
	}

	if o.DatabasePath == "" {
		o.DatabasePath = path.Join(o.DatabaseDir, "bundles.db")
	}

	// create the client
	if o.Client == nil {
		opts := apprclient.Options{Source: o.AppRegistryEndpoint}
		if o.AuthToken != "" {
			opts.AuthToken = o.AuthToken
		}

		client, err := apprclient.New(opts)
		if err != nil {
			return err
		}

		o.Client = client
	}

	return nil
}

// Apply sequentially applies the given options to the config.
func (c *AppregistryBuildOptions) Apply(options []AppregistryBuildOption) {
	for _, option := range options {
		option(c)
	}
}

// ToOption converts an AppregistryBuildOptions object into a function that applies
// its current configuration to another AppregistryBuildOptions instance
func (c *AppregistryBuildOptions) ToOption() AppregistryBuildOption {
	return func(o *AppregistryBuildOptions) {
		if c.Appender != nil {
			o.Appender = c.Appender
		}
		if c.From != "" {
			o.From = c.From
		}
		if c.To != "" {
			o.To = c.To
		}
		if c.AuthToken != "" {
			o.AuthToken = c.AuthToken
		}
		if c.AppRegistryOrg != "" {
			o.AppRegistryOrg = c.AppRegistryOrg
		}
		if c.AppRegistryEndpoint != "" {
			o.AppRegistryEndpoint = c.AppRegistryEndpoint
		}
		if c.DatabasePath != "" {
			o.DatabasePath = c.DatabasePath
		}
		if c.CacheDir != "" {
			o.CacheDir = c.CacheDir
		}
		if c.DatabaseDir != "" {
			o.DatabaseDir = c.DatabaseDir
		}
		if c.Client != nil {
			o.Client = c.Client
		}
	}
}

type AppregistryBuildOption func(*AppregistryBuildOptions)

func DefaultAppregistryBuildOptions() *AppregistryBuildOptions {
	return &AppregistryBuildOptions{
		AppRegistryEndpoint: "https://quay.io/cnr",
		From:                "quay.io/operator-framework/operator-registry-server:latest",
	}
}

func WithAppender(a ImageAppender) AppregistryBuildOption {
	return func(o *AppregistryBuildOptions) {
		o.Appender = a
	}
}

func WithFrom(s string) AppregistryBuildOption {
	return func(o *AppregistryBuildOptions) {
		o.From = s
	}
}

func WithTo(s string) AppregistryBuildOption {
	return func(o *AppregistryBuildOptions) {
		o.To = s
	}
}

func WithAuthToken(s string) AppregistryBuildOption {
	return func(o *AppregistryBuildOptions) {
		o.AuthToken = s
	}
}
func WithAppRegistryEndpoint(s string) AppregistryBuildOption {
	return func(o *AppregistryBuildOptions) {
		o.AppRegistryEndpoint = s
	}
}

func WithAppRegistryOrg(s string) AppregistryBuildOption {
	return func(o *AppregistryBuildOptions) {
		o.AppRegistryOrg = s
	}
}

func WithDatabasePath(s string) AppregistryBuildOption {
	return func(o *AppregistryBuildOptions) {
		o.DatabasePath = s
	}
}

func WithCacheDir(s string) AppregistryBuildOption {
	return func(o *AppregistryBuildOptions) {
		o.CacheDir = s
	}
}

func WithClient(c apprclient.Client) AppregistryBuildOption {
	return func(o *AppregistryBuildOptions) {
		o.Client = c
	}
}
