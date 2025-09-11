package app

import (
	"log/slog"

	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/app/set"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/flag"
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
		Name:  "app",
		Usage: "Commands for apps",
		Commands: []*cli.Command{
			set.New(r.logger, r.version),
		},
		Flags: []cli.Flag{
			flag.LogLevel(),
			flag.Config(),
		},
	}
}
