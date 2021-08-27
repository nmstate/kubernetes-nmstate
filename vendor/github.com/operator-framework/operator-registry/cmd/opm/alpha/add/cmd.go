package add

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/operator-framework/operator-registry/pkg/action"
	"github.com/operator-framework/operator-registry/pkg/image/containerdregistry"
	"github.com/operator-framework/operator-registry/pkg/lib/certs"
)

const (
	retryInterval = time.Second * 5
	timeout       = time.Minute * 1
)

type add struct {
	logger     *logrus.Entry
	configsDir string
	debug      bool
	caFile     string
	pullTool   string
	skipTLS    bool
}

func NewCmd() *cobra.Command {
	logger := logrus.New()
	a := add{
		logger:   logrus.NewEntry(logger),
		pullTool: "none",
	}
	rootCmd := &cobra.Command{
		Use:   "add <configs_path> <bundle_image1> <bundle_image2>........<bundle_imageN>",
		Short: "add operator bundle/s to a catalog of packages",
		Long:  `add operator bundles to a directory of configs representing packages in the catalog`,
		Args:  cobra.MinimumNArgs(2),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			a.configsDir = args[0]
			if a.debug {
				logger.SetLevel(logrus.DebugLevel)
			}
			return nil
		},

		RunE: a.addFunc,
	}

	rootCmd.Flags().BoolVar(&a.debug, "debug", false, "enable debug logging")
	rootCmd.Flags().StringVarP(&a.caFile, "ca-file", "", "", "the root Certificates to use with this command")
	rootCmd.Flags().BoolVar(&a.skipTLS, "skip-tls", false, "disable TLS verification")
	return rootCmd
}

func (a *add) addFunc(cmd *cobra.Command, args []string) error {
	rootCAs, err := certs.RootCAs(a.caFile)
	if err != nil {
		return fmt.Errorf("failed to get RootCAs: %v", err)
	}
	reg, err := containerdregistry.NewRegistry(containerdregistry.SkipTLS(a.skipTLS), containerdregistry.WithLog(a.logger), containerdregistry.WithRootCAs(rootCAs))
	if err != nil {
		return err
	}
	defer func() {
		if err := reg.Destroy(); err != nil {
			a.logger.Errorf("error destroying local cache: %v", err)
		}
	}()
	bundles := []action.BundleExtractor{}
	for _, ref := range args[1:] {
		bundles = append(bundles, action.NewImageBundleExtractor(ref, reg, a.logger))
	}
	request := action.AddConfigRequest{
		Bundles:    bundles,
		ConfigsDir: a.configsDir,
	}
	adder := action.NewBundleAdder(a.logger)
	return adder.AddToConfig(request)
}
