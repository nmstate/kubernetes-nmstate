package alpha

import (
	"github.com/spf13/cobra"

	"github.com/operator-framework/operator-registry/cmd/opm/alpha/bundle"
	"github.com/operator-framework/operator-registry/cmd/opm/alpha/diff"
	"github.com/operator-framework/operator-registry/cmd/opm/alpha/generate"
	"github.com/operator-framework/operator-registry/cmd/opm/alpha/list"
)

func NewCmd() *cobra.Command {
	runCmd := &cobra.Command{
		Hidden: true,
		Use:    "alpha",
		Short:  "Run an alpha subcommand",
		Args:   cobra.NoArgs,
	}

	runCmd.AddCommand(
		bundle.NewCmd(),
		list.NewCmd(),
		generate.NewCmd(),
		diff.NewCmd(),
	)
	return runCmd
}
