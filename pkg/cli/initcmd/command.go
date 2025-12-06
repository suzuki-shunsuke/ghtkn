// Package initcmd implements the 'ghtkn init' command.
// This package is responsible for generating ghtkn configuration files (.ghtkn.yaml)
// with default settings to help users quickly set up ghtkn in their repositories.
// It creates configuration templates that define target workflow files and
// action ignore patterns for the pinning process.
package initcmd

import (
	"context"
	"fmt"

	"github.com/spf13/afero"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/flag"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/initcmd"
	"github.com/suzuki-shunsuke/slog-util/slogutil"
	"github.com/suzuki-shunsuke/urfave-cli-v3-util/urfave"
	"github.com/urfave/cli/v3"
)

// New creates a new init command instance with the provided logger.
// It returns a CLI command that can be registered with the main CLI application.
func New(logger *slogutil.Logger) *cli.Command {
	return Command(logger)
}

// Command returns the CLI command definition for the init subcommand.
// It defines the command name, usage, description, and action handler.
func Command(logger *slogutil.Logger) *cli.Command {
	return &cli.Command{
		Name:   "init",
		Usage:  "Create ghtkn.yaml if it doesn't exist",
		Action: urfave.Action(action, logger),
		Flags: []cli.Flag{
			flag.LogLevel(),
			flag.Config(),
		},
	}
}

func action(_ context.Context, c *cli.Command, logger *slogutil.Logger) error {
	if lvlS := flag.LogLevelValue(c); lvlS != "" {
		if err := logger.SetLevel(lvlS); err != nil {
			return fmt.Errorf("set log level: %w", err)
		}
	}

	configFilePath := c.Args().First()
	if configFilePath == "" {
		configFilePath = flag.ConfigValue(c)
	}
	if configFilePath == "" {
		p, err := ghtkn.GetConfigPath()
		if err != nil {
			return fmt.Errorf("get the config path: %w", err)
		}
		configFilePath = p
	}
	fs := afero.NewOsFs()
	ctrl := initcmd.New(fs)
	return ctrl.Init(logger.Logger, configFilePath) //nolint:wrapcheck
}
