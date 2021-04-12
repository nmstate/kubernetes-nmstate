//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 . ManifestDownloader
package appregistry

import (
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog"

	"github.com/operator-framework/operator-registry/pkg/apprclient"
)

// NewDownloader is a constructor for the Downloader interface
func NewManifestDownloader(client apprclient.Client) ManifestDownloader {
	return &manifestDownloader{
		client: client,
	}
}

// ManifestDownloader is an interface that is implemented by structs that
// implement the DownloadManifests method.
type ManifestDownloader interface {
	// DownloadManifests downloads the manifests in a namespace into a local directory
	DownloadManifests(directory, namespace string) error
}

type manifestDownloader struct {
	client apprclient.Client
}

func (d *manifestDownloader) DownloadManifests(directory, namespace string) error {
	klog.V(4).Infof("Downloading manifests at namespace %s to %s", namespace, directory)

	log := logrus.New().WithField("ns", namespace)

	packages, err := d.client.ListPackages(namespace)
	if err != nil {
		return err
	}

	var errs []error
	for _, pkg := range packages {
		klog.V(4).Infof("Downloading %s", pkg)
		manifest, err := d.client.RetrieveOne(namespace+"/"+pkg.Name, pkg.Release)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		decoder, err := NewManifestDecoder(log)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if _, err = decoder.Decode([]*apprclient.OperatorMetadata{manifest}, directory); err != nil {
			errs = append(errs, err)
			continue
		}
	}

	return errors.NewAggregate(errs)
}
