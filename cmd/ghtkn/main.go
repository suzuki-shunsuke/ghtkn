package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/suzuki-shunsuke/ghtkn/pkg/cli"
	"github.com/suzuki-shunsuke/ghtkn/pkg/log"
	"github.com/suzuki-shunsuke/go-stdutil"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
)

var (
	version = ""
	commit  = "" //nolint:gochecknoglobals
	date    = "" //nolint:gochecknoglobals
)

func main() {
	if code := core(); code != 0 {
		os.Exit(code)
	}
}

func core() int {
	logLevel := &slog.LevelVar{}
	logger := log.New(os.Stderr, version, logLevel)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := cli.Run(ctx, logger, &stdutil.LDFlags{
		Version: version,
		Commit:  commit,
		Date:    date,
	}, logLevel, os.Args...); err != nil {
		slogerr.WithError(logger, err).Error("ghtkn failed")
		return 1
	}
	return 0
}
