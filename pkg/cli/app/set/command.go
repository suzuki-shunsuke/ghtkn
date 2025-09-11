package set

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/flag"
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
		Name:   "set",
		Usage:  "Set a GitHub App Client ID to keyring",
		Action: r.action,
		Flags: []cli.Flag{
			flag.LogLevel(),
			flag.Config(),
		},
	}
}

func (r *runner) action(ctx context.Context, c *cli.Command) error {
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
	if configFilePath == "" {
		p, err := ghtkn.GetConfigPath()
		if err != nil {
			return fmt.Errorf("get the config path: %w", err)
		}
		configFilePath = p
	}
	client := ghtkn.New()
	return client.SetApp(ctx, logger, &ghtkn.InputSetApp{ //nolint:wrapcheck
		ConfigFilePath: configFilePath,
		AppName:        c.Args().First(),
	})
}
