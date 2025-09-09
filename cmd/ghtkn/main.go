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
	logger := log.New(version, slog.LevelInfo)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := cli.Run(ctx, logger, &stdutil.LDFlags{ //nolint:wrapcheck
		Version: version,
		Commit:  commit,
		Date:    date,
	}, os.Args...); err != nil {
		slogerr.WithError(logger, err).Error("ghtkn failed")
		return 1
	}
	return 0
}
