package index

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// AddCommand adds the index subcommand to the given parent command.
func AddCommand(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "index",
		Short: "generate operator index container images",
		Long:  `generate operator index container images from preexisting operator bundles`,

		PreRunE: func(cmd *cobra.Command, args []string) error {
			if debug, _ := cmd.Flags().GetBool("debug"); debug {
				logrus.SetLevel(logrus.DebugLevel)
			}
			return nil
		},
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if skipTLS, err := cmd.Flags().GetBool("skip-tls"); err == nil && skipTLS {
				logrus.Warn("--skip-tls flag is set: this mode is insecure and meant for development purposes only.")
			}
		},
	}

	parent.AddCommand(cmd)
	parent.PersistentFlags().Bool("skip-tls", false, "skip TLS certificate verification for container image registries while pulling bundles or index")
	cmd.AddCommand(newIndexDeleteCmd())
	addIndexAddCmd(cmd)
	cmd.AddCommand(newIndexExportCmd())
	cmd.AddCommand(newIndexPruneCmd())
	cmd.AddCommand(newIndexDeprecateTruncateCmd())
	cmd.AddCommand(newIndexPruneStrandedCmd())
}
