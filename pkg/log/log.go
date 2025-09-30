// Package log provides structured logging functionality for ghtkn.
// It uses slog with tint handler for colored output to stderr.
package log

import (
	"errors"
	"io"
	"log/slog"

	"github.com/lmittmann/tint"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
)

// New creates a new structured logger with the specified version and log level.
// The logger outputs to stderr with colored formatting using tint handler.
// It includes "program" and "version" attributes in all log entries.
func New(w io.Writer, version string) (*slog.Logger, *slog.LevelVar) {
	level := &slog.LevelVar{}
	return slog.New(tint.NewHandler(w, &tint.Options{
		Level: level,
	})).With("program", "ghtkn", "version", version), level
}

// ErrUnknownLogLevel is returned when an invalid log level string is provided to ParseLevel.
var ErrUnknownLogLevel = errors.New("unknown log level")

func SetLevel(logger *slog.Logger, levelVar *slog.LevelVar, level string) {
	lvl, err := parseLevel(level)
	if err != nil {
		slogerr.WithError(logger, err).Warn("parse log level", "level", level)
		return
	}
	levelVar.Set(lvl)
}

// parseLevel converts a string log level to slog.Level.
// Supported levels are: "debug", "info", "warn", "error".
// Returns ErrUnknownLogLevel if the level string is not recognized.
func parseLevel(lvl string) (slog.Level, error) {
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
