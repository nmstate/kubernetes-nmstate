package registry

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/operator-framework/operator-registry/pkg/appregistry"
)

func BuildCmd() *cobra.Command {
	o := appregistry.DefaultAppregistryBuildOptions()
	cmd := &cobra.Command{
		Hidden: true,
		Use:    "build",
		Short:  "build an operator-registry catalog",
		Long:   `build an operator-registry catalog image from other sources`,

		PreRunE: func(cmd *cobra.Command, args []string) error {
			if debug, _ := cmd.Flags().GetBool("debug"); debug {
				logrus.SetLevel(logrus.DebugLevel)
			}
			return nil
		},

		RunE: func(cmd *cobra.Command, args []string) error {
			var noopAppender appregistry.ImageAppendFunc = func(from, to, layer string) error {
				return nil
			}
			builder, err := appregistry.NewAppregistryImageBuilder(o.ToOption(), appregistry.WithAppender(noopAppender))
			if err != nil {
				return err
			}
			return builder.Build()
		},
	}
	flags := cmd.Flags()

	cmd.Flags().Bool("debug", false, "Enable debug logging.")
	flags.StringVar(&o.From, "from", o.From, "The image to use as a base.")
	flags.StringVar(&o.To, "to", "", "The image repository tag to apply to the built catalog image.")
	flags.StringVar(&o.AuthToken, "auth-token", "", "Auth token for communicating with an application registry.")
	flags.StringVar(&o.AppRegistryEndpoint, "appregistry-endpoint", o.AppRegistryEndpoint, "Endpoint for pulling from an application registry instance.")
	flags.StringVarP(&o.AppRegistryOrg, "appregistry-org", "o", "", "Organization (Namespace) to pull from an application registry instance")
	flags.StringVarP(&o.DatabasePath, "to-db", "d", "", "Local path to save the database to.")
	flags.StringVarP(&o.CacheDir, "dir", "c", "", "Local path to cache manifests when downloading.")

	return cmd
}
