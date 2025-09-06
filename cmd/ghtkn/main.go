package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/log"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli"
	"github.com/suzuki-shunsuke/go-stdutil"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
)

var (
	version = ""
	commit  = "" //nolint:gochecknoglobals
	date    = "" //nolint:gochecknoglobals
)

func main() {
	logger := log.New(version, slog.LevelInfo)
	if err := core(logger); err != nil {
		slogerr.WithError(logger, err).Error("ghtkn failed")
		os.Exit(1)
	}
}

func core(logger *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return cli.Run(ctx, logger, &stdutil.LDFlags{ //nolint:wrapcheck
		Version: version,
		Commit:  commit,
		Date:    date,
	}, os.Args...)
}
