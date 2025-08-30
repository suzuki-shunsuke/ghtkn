package log

import (
	"errors"
	"log/slog"
	"os"

	"github.com/lmittmann/tint"
)

func New(version string, level slog.Level) *slog.Logger {
	return slog.New(tint.NewHandler(os.Stderr, &tint.Options{
		Level: level,
	})).With("program", "ghtkn", "version", version)
}

var ErrUnknownLogLevel = errors.New("unknown log level")

func ParseLevel(lvl string) (slog.Level, error) {
	switch lvl {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, ErrUnknownLogLevel
	}
}
