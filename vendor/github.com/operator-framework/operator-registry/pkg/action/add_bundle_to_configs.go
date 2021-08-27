package action

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/otiai10/copy"
	"github.com/sirupsen/logrus"

	"github.com/operator-framework/operator-registry/internal/declcfg"
	"github.com/operator-framework/operator-registry/pkg/registry"
)

type AddConfigRequest struct {
	ConfigsDir string
	Bundles    []BundleExtractor
}

type BundleAdder struct {
	Logger *logrus.Entry
}

func NewBundleAdder(logger *logrus.Entry) BundleAdder {
	return BundleAdder{
		Logger: logger,
	}
}

func (b BundleAdder) AddToConfig(request AddConfigRequest) error {
	b.Logger.Infof("loading configs from directory")
	decCfg, err := declcfg.LoadDir(request.ConfigsDir)
	if err != nil {
		return fmt.Errorf("error loading directory %q: %v", request.ConfigsDir, err)
	}

	model, err := declcfg.ConvertToModel(*decCfg)
	if err != nil {
		return fmt.Errorf("error converting configs to internal model:%v", err)
	}

	for _, bundle := range request.Bundles {
		b, err := bundle.ExtractBundle(context.TODO())
		mBundles, err := registry.ConvertRegistryBundleToModelBundles(b)
		if err != nil {
			return fmt.Errorf("error creating internal model bundles from registry bundle %q: %v", b.BundleImage, err)
		}
		for _, b := range mBundles {
			model.AddBundle(b)
		}
	}

	newDecCfg := declcfg.ConvertFromModel(model)
	tmpDir, err := ioutil.TempDir("./", "configs_tmp")
	if err != nil {
		return fmt.Errorf("error creating temp directory to write modified configs into:%v", err)
	}

	b.Logger.Infof("writing modified temp directory to %q", tmpDir)
	err = declcfg.WriteDir(newDecCfg, tmpDir)

	if err != nil {
		return fmt.Errorf("error writing new configs into %q:%v", tmpDir, err)
	} else {
		if err := os.RemoveAll(request.ConfigsDir); err != nil { // this is required because WriteDir only writes to an empty directory
			return fmt.Errorf("could not remove existing output dir %q: %v", request.ConfigsDir, err)
		}
		if err := os.Mkdir(request.ConfigsDir, os.ModePerm); err != nil {
			return fmt.Errorf("could not create new directory %q: %v", request.ConfigsDir, err)
		}
		b.Logger.Infof("rewriting contents of existing configs directory %q with contents from %q", request.ConfigsDir, tmpDir)
		err = copy.Copy(tmpDir, request.ConfigsDir)
		if err != nil {
			return fmt.Errorf("error copying new configs to %q:%v", request.ConfigsDir, err)
		}
	}
	return os.RemoveAll(tmpDir)
}

func existingPackage(decCfg *declcfg.DeclarativeConfig, pkg string) bool {
	for _, p := range decCfg.Packages {
		if p.Name == pkg {
			return true
		}
	}
	return false
}
