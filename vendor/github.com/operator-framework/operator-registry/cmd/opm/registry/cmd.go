package registry

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// NewOpmRegistryCmd returns the appregistry-server command
func NewOpmRegistryCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "registry",
		Short: "interact with operator-registry database",
		Long:  `interact with operator-registry database building, modifying and/or serving the operator-registry database`,

		PreRunE: func(cmd *cobra.Command, args []string) error {
			if debug, _ := cmd.Flags().GetBool("debug"); debug {
				logrus.SetLevel(logrus.DebugLevel)
			}
			return nil
		},
	}

	rootCmd.AddCommand(newRegistryServeCmd())
	rootCmd.AddCommand(newRegistryAddCmd())
	rootCmd.AddCommand(newRegistryRmCmd())
	rootCmd.AddCommand(newRegistryPruneCmd())
	rootCmd.AddCommand(newRegistryPruneStrandedCmd())

	return rootCmd
}
