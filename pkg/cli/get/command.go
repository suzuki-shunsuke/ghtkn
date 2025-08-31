// Package get implements the 'ghtkn get' command.
// This command retrieves or creates GitHub App User Access Tokens and outputs them to stdout.
// It handles token persistence, expiration checking, and automatic renewal when needed.
package get

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/flag"
	"github.com/suzuki-shunsuke/ghtkn/pkg/config"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/get"
	"github.com/suzuki-shunsuke/ghtkn/pkg/log"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
	"github.com/urfave/cli/v3"
)

// New creates a new get command instance with the provided logger and version.
// It returns a CLI command that can be registered with the main CLI application.
func New(logger *slog.Logger, version string) *cli.Command {
	r := &runner{
		logger:  logger,
		version: version,
	}
	return r.Command()
}

// runner encapsulates the state and behavior for the get command.
type runner struct {
	logger  *slog.Logger
	version string
}

// Command returns the CLI command definition for the get subcommand.
// It defines the command name, usage, action handler, and available flags.
func (r *runner) Command() *cli.Command {
	return &cli.Command{
		Name:   "get",
		Usage:  "Output a GitHub App User Access Token to stdout",
		Action: r.action,
		Flags: []cli.Flag{
			flag.LogLevel(),
			flag.Config(),
			flag.Format(),
			flag.MinExpiration(),
		},
	}
}

// action implements the main logic for the get command.
// It configures the controller with flags and arguments, then executes the token retrieval.
// Returns an error if configuration is invalid or token retrieval fails.
func (r *runner) action(ctx context.Context, c *cli.Command) error {
	logger := r.logger
	if lvlS := flag.LogLevelValue(c); lvlS != "" {
		lvl, err := log.ParseLevel(lvlS)
		if err != nil {
			return fmt.Errorf("parse the log level: %w", slogerr.With(err, "log_level", lvlS))
		}
		logger = log.New(r.version, lvl)
	}
	input := get.NewInput(flag.ConfigValue(c))
	if input.ConfigFilePath == "" {
		input.ConfigFilePath = config.GetPath(input.Env)
	}
	input.OutputFormat = flag.FormatValue(c)
	if m := flag.MinExpirationValue(c); m != "" {
		d, err := time.ParseDuration(m)
		if err != nil {
			return fmt.Errorf("parse the min expiration: %w", slogerr.With(err, "min_expiration", m))
		}
		input.MinExpiration = d
	}
	if err := input.Validate(); err != nil {
		return err //nolint:wrapcheck
	}
	if arg := c.Args().First(); arg != "" {
		input.Env.App = arg
	}
	return get.New(input).Run(ctx, logger) //nolint:wrapcheck
}
