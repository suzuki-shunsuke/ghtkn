// Package initcmd implements the 'ghtkn init' command.
// This package is responsible for generating ghtkn configuration files (.ghtkn.yaml)
// with default settings to help users quickly set up ghtkn in their repositories.
// It creates configuration templates that define target workflow files and
// action ignore patterns for the pinning process.
package initcmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"

	"github.com/spf13/afero"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/flag"
	"github.com/suzuki-shunsuke/ghtkn/pkg/config"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/initcmd"
	"github.com/suzuki-shunsuke/ghtkn/pkg/log"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
	"github.com/urfave/cli/v3"
)

// New creates a new init command instance with the provided logger.
// It returns a CLI command that can be registered with the main CLI application.
func New(logger *slog.Logger, version string) *cli.Command {
	r := &runner{
		logger:  logger,
		version: version,
	}
	return r.Command()
}

type runner struct {
	logger  *slog.Logger
	version string
}

// Command returns the CLI command definition for the init subcommand.
// It defines the command name, usage, description, and action handler.
func (r *runner) Command() *cli.Command {
	return &cli.Command{
		Name:   "init",
		Usage:  "Create ghtkn.yaml if it doesn't exist",
		Action: r.action,
		Flags: []cli.Flag{
			flag.LogLevel(),
			flag.Config(),
		},
	}
}

func (r *runner) action(_ context.Context, c *cli.Command) error {
	logger := r.logger
	if lvlS := flag.LogLevelValue(c); lvlS != "" {
		lvl, err := log.ParseLevel(lvlS)
		if err != nil {
			return slogerr.With(err, "log_level", lvlS) //nolint:wrapcheck
		}
		logger = log.New(r.version, lvl)
	}

	configFilePath := c.Args().First()
	if configFilePath == "" {
		configFilePath = flag.ConfigValue(c)
	}
	env := config.NewEnv(os.Getenv, runtime.GOOS)
	if configFilePath == "" {
		p, err := config.GetPath(env)
		if err != nil {
			return fmt.Errorf("get the config path: %w", err)
		}
		configFilePath = p
	}
	fs := afero.NewOsFs()
	ctrl := initcmd.New(fs, env)
	return ctrl.Init(logger, configFilePath) //nolint:wrapcheck
}
