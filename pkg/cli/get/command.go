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
