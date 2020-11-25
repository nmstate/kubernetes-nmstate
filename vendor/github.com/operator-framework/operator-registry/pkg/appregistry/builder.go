//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 builder.go ImageAppender
package appregistry

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"database/sql"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"k8s.io/klog"

	"github.com/operator-framework/operator-registry/pkg/apprclient"
	"github.com/operator-framework/operator-registry/pkg/sqlite"
)

type RegistryImageBuilder interface {
	Build() error
}

type ImageAppender interface {
	Append(from, to, layer string) error
}

type ImageAppendFunc func(from, to, layer string) error

func (f ImageAppendFunc) Append(from, to, layer string) error {
	return f(from, to, layer)
}

type AppregistryImageBuilder struct {
	Appender ImageAppender

	// options
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
	client      apprclient.Client
}

func NewAppregistryImageBuilder(options ...AppregistryBuildOption) (*AppregistryImageBuilder, error) {
	config := DefaultAppregistryBuildOptions()
	config.Apply(options)
	if err := config.Complete(); err != nil {
		return nil, err
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return &AppregistryImageBuilder{
		Appender:            config.Appender,
		From:                config.From,
		To:                  config.To,
		AuthToken:           config.AuthToken,
		AppRegistryEndpoint: config.AppRegistryEndpoint,
		AppRegistryOrg:      config.AppRegistryOrg,
		DatabasePath:        config.DatabasePath,
		CacheDir:            config.CacheDir,
		CleanOutput:         config.CleanOutput,
		ManifestDir:         config.ManifestDir,
		DatabaseDir:         config.DatabaseDir,
		client:              config.Client,
	}, nil
}

func (b *AppregistryImageBuilder) Build() error {

	defer func() {
		if !b.CleanOutput {
			return
		}
		if err := os.RemoveAll(b.CacheDir); err != nil {
			klog.Warningf("unable to clean %s", b.CacheDir)
		}
	}()

	downloader := NewManifestDownloader(b.client)
	if err := downloader.DownloadManifests(b.ManifestDir, b.AppRegistryOrg); err != nil {
		return err
	}

	if !hasManifests(b.ManifestDir) {
		return fmt.Errorf("no manifests downloaded from appregistry %s/%s", b.AppRegistryEndpoint, b.AppRegistryOrg)
	}

	klog.V(4).Infof("downloaded manifests to %s\n", b.ManifestDir)

	if err := BuildDatabase(b.ManifestDir, b.DatabasePath); err != nil {
		return err
	}

	klog.V(4).Infof("database written %s\n", b.DatabasePath)

	if b.To == "" {
		return nil
	}

	archivePath, err := BuildLayer(b.DatabaseDir)
	if err != nil {
		return err
	}
	klog.V(4).Infof("built db layer %s\n", archivePath)

	return b.Appender.Append(b.From, b.To, archivePath)
}

func BuildDatabase(manifestPath, databasePath string) error {
	db, err := sql.Open("sqlite3", databasePath)
	if err != nil {
		return err
	}
	dbLoader, err := sqlite.NewSQLLiteLoader(db)
	if err != nil {
		return err
	}
	defer func() {
		if err := db.Close(); err != nil {
			klog.Warningf(err.Error())
		}
	}()

	if err := dbLoader.Migrate(context.TODO()); err != nil {
		return err
	}

	loader := sqlite.NewSQLLoaderForDirectory(dbLoader, manifestPath)
	if err := loader.Populate(); err != nil {
		klog.Warningf("error building database: %s", err.Error())
	}
	return nil
}

func BuildLayer(directory string) (string, error) {
	archiveDir, err := ioutil.TempDir("", "archive-")
	if err != nil {
		return "", err
	}

	archive, err := os.Create(path.Join(archiveDir, "layer.tar.gz"))
	if err != nil {
		return "", err
	}
	defer func() {
		if err := archive.Close(); err != nil {
			klog.Warningf("error closing file: %s", err.Error())
		}
	}()

	gzipWriter := gzip.NewWriter(archive)
	defer func() {
		if err := gzipWriter.Close(); err != nil {
			klog.Warningf("error closing writer: %s", err.Error())
		}
	}()
	writer := tar.NewWriter(gzipWriter)
	defer func() {
		if err := writer.Close(); err != nil {
			klog.Warningf("error closing writer: %s", err.Error())
		}
	}()

	if err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.Mode().IsRegular() {
			return nil
		}

		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}

		err = writer.WriteHeader(header)
		if err != nil {
			return err
		}

		// if it's a directory, just write the header and continue
		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer func() {
			if err := file.Close(); err != nil {
				klog.Warningf("error closing file: %s", err.Error())
			}
		}()

		_, err = io.Copy(writer, file)
		if err != nil {
			return err
		}

		return err
	}); err != nil {
		return "", err
	}

	return archive.Name(), nil
}

func hasManifests(path string) bool {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return false
	}
	return len(files) > 0
}
