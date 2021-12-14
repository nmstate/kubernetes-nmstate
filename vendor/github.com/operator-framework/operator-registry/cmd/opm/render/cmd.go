package render

import (
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/operator-framework/operator-registry/alpha/action"
	"github.com/operator-framework/operator-registry/alpha/declcfg"
	"github.com/operator-framework/operator-registry/pkg/sqlite"
)

func NewCmd() *cobra.Command {
	var (
		render action.Render
		output string
	)
	cmd := &cobra.Command{
		Use:   "render [index-image | bundle-image | sqlite-file]...",
		Short: "Generate a declarative config blob from catalogs and bundles",
		Long: `Generate a declarative config blob from the provided index images, bundle images, and sqlite database files

` + sqlite.DeprecationMessage,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			render.Refs = args

			var write func(declcfg.DeclarativeConfig, io.Writer) error
			switch output {
			case "yaml":
				write = declcfg.WriteYAML
			case "json":
				write = declcfg.WriteJSON
			default:
				log.Fatalf("invalid --output value %q, expected (json|yaml)", output)
			}

			// The bundle loading impl is somewhat verbose, even on the happy path,
			// so discard all logrus default logger logs. Any important failures will be
			// returned from render.Run and logged as fatal errors.
			logrus.SetOutput(ioutil.Discard)

			cfg, err := render.Run(cmd.Context())
			if err != nil {
				log.Fatal(err)
			}

			if err := write(*cfg, os.Stdout); err != nil {
				log.Fatal(err)
			}
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", "json", "Output format (json|yaml)")
	return cmd
}
