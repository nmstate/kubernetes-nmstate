package appregistry

import (
	"fmt"

	"github.com/sirupsen/logrus"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"

	"github.com/operator-framework/operator-registry/pkg/client"
	"github.com/operator-framework/operator-registry/pkg/registry"
)

// NewLoader returns a new instance of AppregistryLoader.
//
// kubeconfig specifies the location of kube configuration file.
// dbName specifies the database name to be used for sqlite.
// downloadPath specifies the folder where the downloaded nested bundle(s) will
// be stored.
func NewLoader(kubeconfig string, dbName string, downloadPath string, logger *logrus.Entry) (*AppregistryLoader, error) {
	kubeClient, err := client.NewKubeClient(kubeconfig, logger.Logger)
	if err != nil {
		return nil, err
	}

	loader, err := NewDbLoader(dbName, logger)
	if err != nil {
		return nil, err
	}

	specifier := &registrySpecifier{}
	decoder, err := NewManifestDecoder(logger)
	if err != nil {
		return nil, err
	}

	return &AppregistryLoader{
		logger: logger,
		input: &inputParser{
			sourceSpecifier: specifier,
		},
		downloader:   newDownloader(logger, kubeClient),
		downloadPath: downloadPath,
		decoder:      decoder,
		loader:       loader,
	}, nil
}

type AppregistryLoader struct {
	logger       *logrus.Entry
	input        *inputParser
	downloader   *downloader
	downloadPath string
	decoder      *manifestDecoder
	loader       *dbLoader
}

func (a *AppregistryLoader) Load(csvSources []string, csvPackages string) (registry.Query, error) {
	a.logger.Infof("operator source(s) specified are - %s", csvSources)
	a.logger.Infof("package(s) specified are - %s", csvPackages)

	var errs []error
	input, err := a.input.Parse(csvSources, csvPackages)
	if err != nil {
		errs = append(errs, fmt.Errorf("error parsing input: %s", err))
		if input == nil || !input.IsGoodToProceed() {
			a.logger.Info("can't proceed, bailing out")
			return nil, utilerrors.NewAggregate(errs)
		}
	}

	a.logger.Info("input has been sanitized")
	a.logger.Infof("sources: %s", input.Sources)
	a.logger.Infof("packages: %s", input.Packages)

	rawManifests, err := a.downloader.Download(input)
	if err != nil {
		errs = append(errs, fmt.Errorf("error downloading manifests: %s", err))
	}

	a.logger.Infof("download complete - %d repositories have been downloaded", len(rawManifests))

	// The set of operator manifest(s) downloaded is a collection of both
	// flattened single file yaml and nested operator bundle(s).
	result, err := a.decoder.Decode(rawManifests, a.downloadPath)
	if err != nil {
		errs = append(errs, fmt.Errorf("error decoding manifest: %s", err))
	}
	if result.IsEmpty() {
		a.logger.Info("No operator manifest decoded")
	}

	a.logger.Infof("decoded %d flattened and %d nested operator manifest(s)", result.FlattenedCount, result.NestedCount)

	if err = a.loader.LoadBundleDirectoryToSQLite(a.downloadPath); err != nil {
		errs = append(errs, fmt.Errorf("error loading operator manifests: %s", err))
	}

	store := a.loader.GetStore()

	return store, utilerrors.NewAggregate(errs)
}
