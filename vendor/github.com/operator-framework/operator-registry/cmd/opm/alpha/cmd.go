package alpha

import (
	"github.com/operator-framework/operator-registry/cmd/opm/alpha/add"
	"github.com/operator-framework/operator-registry/cmd/opm/alpha/bundle"
	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	runCmd := &cobra.Command{
		Hidden: true,
		Use:    "alpha",
		Short:  "Run an alpha subcommand",
	}

	runCmd.AddCommand(bundle.NewCmd(), add.NewCmd())
	return runCmd
}
