package root

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/operator-framework/operator-registry/cmd/opm/alpha"
	"github.com/operator-framework/operator-registry/cmd/opm/index"
	"github.com/operator-framework/operator-registry/cmd/opm/registry"
	"github.com/operator-framework/operator-registry/cmd/opm/serve"
	"github.com/operator-framework/operator-registry/cmd/opm/validate"
	"github.com/operator-framework/operator-registry/cmd/opm/version"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "opm",
		Short: "operator package manager",
		Long:  "CLI to interact with operator-registry and build indexes of operator content",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if debug, _ := cmd.Flags().GetBool("debug"); debug {
				logrus.SetLevel(logrus.DebugLevel)
			}
			return nil
		},
	}

	cmd.AddCommand(registry.NewOpmRegistryCmd(), alpha.NewCmd(), serve.NewCmd(), validate.NewCmd())
	index.AddCommand(cmd)
	version.AddCommand(cmd)

	cmd.Flags().Bool("debug", false, "enable debug logging")
	if err := cmd.Flags().MarkHidden("debug"); err != nil {
		logrus.Panic(err.Error())
	}

	return cmd
}
