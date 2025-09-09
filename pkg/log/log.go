// Package log provides structured logging functionality for ghtkn.
// It uses slog with tint handler for colored output to stderr.
package log

import (
	"errors"
	"log/slog"
	"os"

	"github.com/lmittmann/tint"
)

// New creates a new structured logger with the specified version and log level.
// The logger outputs to stderr with colored formatting using tint handler.
// It includes "program" and "version" attributes in all log entries.
func New(version string, level slog.Level) *slog.Logger {
	return slog.New(tint.NewHandler(os.Stderr, &tint.Options{
		Level: level,
	})).With("program", "ghtkn", "version", version)
}

// ErrUnknownLogLevel is returned when an invalid log level string is provided to ParseLevel.
var ErrUnknownLogLevel = errors.New("unknown log level")

// ParseLevel converts a string log level to slog.Level.
// Supported levels are: "debug", "info", "warn", "error".
// Returns ErrUnknownLogLevel if the level string is not recognized.
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
