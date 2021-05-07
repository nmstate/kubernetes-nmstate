package action

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"

	"github.com/operator-framework/operator-registry/pkg/image"
	"github.com/operator-framework/operator-registry/pkg/registry"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/util/retry"
)

type BundleExtractor interface {
	ExtractBundle(ctx context.Context) (*registry.Bundle, error)
}

type imageBundleExtractor struct {
	imgRef string
	reg    image.Registry
	logger *logrus.Entry
}

func NewImageBundleExtractor(ref string, reg image.Registry, logger *logrus.Entry) imageBundleExtractor {
	return imageBundleExtractor{
		imgRef: ref,
		reg:    reg,
		logger: logger,
	}
}

func (i imageBundleExtractor) ExtractBundle(ctx context.Context) (*registry.Bundle, error) {

	simpleRef := image.SimpleReference(i.imgRef)
	tmpDir, err := ioutil.TempDir("./", "bundle_tmp")
	if err != nil {
		return nil, fmt.Errorf("error creating temp directory to unpack bundle image %q in:%v", simpleRef.String(), err)
	}
	defer func() {
		i.logger.Infof("Removing temp directory %q bundle was unpacked in", tmpDir)
		if err := os.RemoveAll(tmpDir); err != nil {
			i.logger.Errorf("error removing temp directory %q bundle was unpacked in: %v", tmpDir, err)
		}
	}()
	nonRetryableRegex := regexp.MustCompile(`(error resolving name)`)
	i.logger.Infof("Pulling bundle %q", simpleRef.String())
	if err := retry.OnError(retry.DefaultRetry,
		func(err error) bool {
			if nonRetryableRegex.MatchString(err.Error()) {
				return false
			}
			i.logger.Warnf("  Error pulling image: %v. Retrying.", err)
			return true
		},
		func() error { return i.reg.Pull(ctx, simpleRef) }); err != nil {
		return nil, fmt.Errorf("error pulling image %q into registry:%v", simpleRef.String(), err)
	}
	i.logger.Infof("Unpacking bundle %q into %q", simpleRef.String(), tmpDir)
	err = i.reg.Unpack(ctx, simpleRef, tmpDir)
	if err != nil {
		return nil, fmt.Errorf("error unpacking image %q: %v", simpleRef.String(), err)
	}
	img, err := registry.NewImageInput(simpleRef, tmpDir)
	if err != nil {
		return nil, fmt.Errorf("error interpreting bundle image %q: %v", simpleRef.String(), err)
	}
	return img.Bundle, nil
}
