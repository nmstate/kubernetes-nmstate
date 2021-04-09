package registry

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/operator-framework/operator-registry/pkg/containertools"
	"github.com/operator-framework/operator-registry/pkg/lib/registry"
	reg "github.com/operator-framework/operator-registry/pkg/registry"
)

func newRegistryAddCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "add",
		Short: "add operator bundle to operator registry DB",
		Long:  `add operator bundle to operator registry DB`,

		PreRunE: func(cmd *cobra.Command, args []string) error {
			if debug, _ := cmd.Flags().GetBool("debug"); debug {
				logrus.SetLevel(logrus.DebugLevel)
			}
			return nil
		},

		RunE: addFunc,
	}

	rootCmd.Flags().Bool("debug", false, "enable debug logging")
	rootCmd.Flags().StringP("database", "d", "bundles.db", "relative path to database file")
	rootCmd.Flags().StringSliceP("bundle-images", "b", []string{}, "comma separated list of links to bundle image")
	rootCmd.Flags().Bool("permissive", false, "allow registry load errors")
	rootCmd.Flags().Bool("skip-tls", false, "skip TLS certificate verification for container image registries while pulling bundles")
	rootCmd.Flags().StringP("mode", "", "replaces", "graph update mode that defines how channel graphs are updated. One of: [replaces, semver, semver-skippatch]")
	rootCmd.Flags().StringP("container-tool", "c", "none", "tool to interact with container images (save, build, etc.). One of: [none, docker, podman]")

	return rootCmd
}

func addFunc(cmd *cobra.Command, args []string) error {
	permissive, err := cmd.Flags().GetBool("permissive")
	if err != nil {
		return err
	}
	skipTLS, err := cmd.Flags().GetBool("skip-tls")
	if err != nil {
		return err
	}
	fromFilename, err := cmd.Flags().GetString("database")
	if err != nil {
		return err
	}
	bundleImages, err := cmd.Flags().GetStringSlice("bundle-images")
	if err != nil {
		return err
	}
	containerTool, err := cmd.Flags().GetString("container-tool")
	if err != nil {
		return err
	}
	mode, err := cmd.Flags().GetString("mode")
	if err != nil {
		return err
	}
	modeEnum, err := reg.GetModeFromString(mode)
	if err != nil {
		return err
	}

	request := registry.AddToRegistryRequest{
		Permissive:    permissive,
		SkipTLS:       skipTLS,
		InputDatabase: fromFilename,
		Bundles:       bundleImages,
		Mode:          modeEnum,
		ContainerTool: containertools.NewContainerTool(containerTool, containertools.NoneTool),
		Overwrite:     false,
	}

	logger := logrus.WithFields(logrus.Fields{"bundles": bundleImages})

	if skipTLS {
		logger.Warn("--skip-tls flag is set: this mode is insecure and meant for development purposes only.")
	}

	logger.Info("adding to the registry")

	registryAdder := registry.NewRegistryAdder(logger)

	err = registryAdder.AddToRegistry(request)
	if err != nil {
		return err
	}
	return nil
}
