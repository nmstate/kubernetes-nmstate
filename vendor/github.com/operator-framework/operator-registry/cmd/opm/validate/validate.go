package validate

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/operator-framework/operator-registry/pkg/lib/config"
)

func NewCmd() *cobra.Command {
	validate := &cobra.Command{
		Use:   "validate <directory>",
		Short: "Validate the declarative index config",
		Long:  "Validate the declarative config JSON file(s) in a given directory",
		Args:  cobra.ExactArgs(1),
		RunE:  configValidate,
	}

	validate.Flags().BoolP("debug", "d", false, "enable debug log output")
	return validate
}

func configValidate(cmd *cobra.Command, args []string) error {
	debug, err := cmd.Flags().GetBool("debug")
	if err != nil {
		return err
	}

	directory := args[0]
	if _, err := os.Stat(directory); os.IsNotExist(err) {
		return err
	}

	logger := logrus.WithField("cmd", "validate")
	if debug {
		logger.Logger.SetLevel(logrus.DebugLevel)
	}

	err = config.ValidateConfig(directory)
	if err != nil {
		logger.Error(err.Error())
		return fmt.Errorf("failed to validate config: %s", err)
	}

	return nil
}
