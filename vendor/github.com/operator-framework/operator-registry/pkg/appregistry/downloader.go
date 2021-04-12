package appregistry

import (
	"context"
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/kubernetes"

	"github.com/operator-framework/operator-registry/pkg/apprclient"
)

// downloadItem encapsulates the data that is needed to download a specific repository.
type downloadItem struct {
	// Repository points to the repository and the particular release that needs
	// to be downloaded.
	RepositoryMetadata *apprclient.RegistryMetadata

	// Source refers to the remote appregistry URL and remote registry namespace.
	Source *Source

	// Release refers to the release number the user requested
	Release string
}

func (d *downloadItem) String() string {
	return fmt.Sprintf("%s:%s", d.RepositoryMetadata.Name, d.Release)
}

type registryOptionsGetter interface {
	GetRegistryOptions(source *Source) (*apprclient.Options, error)
}

type secretRegistryOptionsGetter struct {
	kubeClient kubernetes.Interface
}

type sourceQuerier interface {
	QuerySource(source *Source) ([]*apprclient.RegistryMetadata, error)
}

type appRegistrySourceQuerier struct {
	kubeClient      kubernetes.Interface
	regOptionGetter registryOptionsGetter
}

type downloader struct {
	logger          *logrus.Entry
	kubeClient      kubernetes.Interface
	querier         sourceQuerier
	regOptionGetter registryOptionsGetter
}

// NewDownloader returns a new instance of downloader
func newDownloader(logger *logrus.Entry, kubeClient kubernetes.Interface) *downloader {
	regOptionGetter := &secretRegistryOptionsGetter{kubeClient}
	return &downloader{
		logger,
		kubeClient,
		&appRegistrySourceQuerier{kubeClient, regOptionGetter},
		regOptionGetter,
	}
}

// Download downloads manifest(s) associated with the specified package(s) from
// the corresponding operator source(s).
//
// We take a best effort approach in downloading.
//
// If an OperatorSource is not found, we skip it and move on to the next source.
func (d *downloader) Download(input *Input) (manifests []*apprclient.OperatorMetadata, err error) {
	items, err := d.Prepare(input)
	if err != nil {
		d.logger.Errorf("the following error(s) occurred while preparing the download list: %v", err)

		if len(items) == 0 {
			d.logger.Infof("download list is empty, bailing out: %v", input.Packages)
			return
		}
	}

	for _, item := range items {
		d.logger.Infof(
			"the following releases are available for package %s -> %s",
			item.RepositoryMetadata.Name,
			item.RepositoryMetadata.Releases,
		)
	}
	d.logger.Infof("resolved the following packages: %s", items)

	manifests, err = d.DownloadRepositories(items)

	return
}

// Prepare prepares the list of repositories to download by resolving each
// package specified to its corresponding operator source.
//
// If a package is specified more than once, the operator source that it
// resolves to the first time is picked.
//
// We apply a best-effort approach here, if a package is can't be resolved we
// log it and move on.
func (d *downloader) Prepare(input *Input) (items []*downloadItem, err error) {
	packageMap := input.PackagesToMap()
	itemMap := map[Package]*downloadItem{}
	allErrors := []error{}

	for _, source := range input.Sources {
		if len(packageMap) == 0 {
			// All specified package(s) have already been resolved.
			break
		}

		repositoryList, err := d.querier.QuerySource(source)
		if err != nil {
			allErrors = append(allErrors, err)
			d.logger.Infof("skipping operator source due to error: %s", source)

			continue
		}

		repositoryMap := map[string]*apprclient.RegistryMetadata{}
		for _, metadata := range repositoryList {
			repositoryMap[metadata.Name] = metadata
		}

		for _, pkg := range input.Packages {
			metadata, ok := repositoryMap[pkg.Name]
			if !ok {
				// The package is not in the current source
				continue
			}
			// If a specific release was requrested, download it
			release := pkg.Release
			if release != "" {
				releaseMap := metadata.ReleaseMap()
				if _, ok := releaseMap[pkg.Release]; !ok {
					// We have the package, but not the requested release
					continue
				}
			} else {
				// default to the latest
				release = metadata.Release
			}

			itemMap[*pkg] = &downloadItem{
				RepositoryMetadata: metadata,
				Release:            release,
				Source:             source,
			}
			delete(packageMap, *pkg)
		}
	}

	// We might still have packages specified that have not been resolved.
	if len(packageMap) > 0 {
		d.logger.Infof("the following packages could not be resolved: %v", packageMap)
	}

	items = make([]*downloadItem, 0)
	for _, v := range itemMap {
		items = append(items, v)
	}

	err = utilerrors.NewAggregate(allErrors)
	return
}

// DownloadRepositories iterates through each download item and downloads
// operator manifest from the corresponding repository.
func (d *downloader) DownloadRepositories(items []*downloadItem) (manifests []*apprclient.OperatorMetadata, err error) {
	allErrors := []error{}

	manifests = make([]*apprclient.OperatorMetadata, 0)
	for _, item := range items {
		endpoint := item.Source.Endpoint

		d.logger.Infof("downloading repository: %s from %s", item, endpoint)

		options, err := d.regOptionGetter.GetRegistryOptions(item.Source)
		if err != nil {
			allErrors = append(allErrors, err)
			d.logger.Infof("skipping repository: %s", item)

			continue
		}

		client, err := apprclient.New(*options)
		if err != nil {
			allErrors = append(allErrors, err)
			d.logger.Infof("skipping repository: %s", item)

			continue
		}

		manifest, err := client.RetrieveOne(item.RepositoryMetadata.ID(), item.Release)
		if err != nil {
			allErrors = append(allErrors, err)
			d.logger.Infof("skipping repository: %s", item)

			continue
		}

		manifests = append(manifests, manifest)
	}

	err = utilerrors.NewAggregate(allErrors)
	return
}

// QuerySource retrives the OperatorSource object specified by key. It queries
// the registry namespace to list all the repositories associated with this
// operator source.
//
// The function returns the spec ( associated with the OperatorSource object )
// in the cluster and the list of repositories in remote registry associated
// with it.
func (a *appRegistrySourceQuerier) QuerySource(source *Source) (repositories []*apprclient.RegistryMetadata, err error) {
	if source == nil {
		return nil, errors.New("specified source is <nil>")
	}

	options, err := a.regOptionGetter.GetRegistryOptions(source)
	if err != nil {
		return
	}

	client, err := apprclient.New(*options)
	if err != nil {
		return
	}

	repositories, err = client.ListPackages(source.RegistryNamespace)
	if err != nil {
		return
	}

	return
}

// GetRegistryOptions generates an Options object based on the OperatorSource spec. It passes along
// the opsrc endpoint and, if defined, retrieves the authorization token from the specified Secret
// object.
func (s *secretRegistryOptionsGetter) GetRegistryOptions(source *Source) (*apprclient.Options, error) {
	if source == nil {
		return nil, errors.New("specified source is <nil>")
	}

	token := ""
	if source.IsSecretSpecified() {
		secret, err := s.kubeClient.CoreV1().Secrets(source.Secret.Namespace).Get(context.TODO(), source.Secret.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

		token = string(secret.Data["token"])
	}

	options := &apprclient.Options{
		Source:    source.Endpoint,
		AuthToken: token,
	}

	return options, nil
}
