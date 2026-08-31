// Package flag provides common command-line flags for ghtkn CLI.
// It defines reusable flag definitions for consistent flag handling across all commands.
//
// Each function registers its flag on the given flag set rather than returning a
// definition, because that is how pflag works; the destination pointer is where the
// value lands, as it was with urfave/cli.
package flag

import (
	"fmt"
	"time"

	"github.com/spf13/pflag"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/env"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cobrautil"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
)

// GlobalFlags holds the global flag values for the root command.
type GlobalFlags struct {
	LogLevel string
	Config   string
}

// SetMinExpiration parses the --min-expiration flag value and sets it on inputGet.
// It lives next to the flag definition because the two share the precedence the
// comment on MinExpiration describes, and both 'get' and 'exec' accept the flag.
// When the flag is not set it leaves inputGet.MinExpiration nil, so the SDK falls
// back to GHTKN_MIN_EXPIRATION and the config; an explicit value, including 0, takes
// precedence over both.
func SetMinExpiration(inputGet *ghtkn.InputGet, s string) error {
	if s == "" {
		return nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("parse the min expiration: %w", slogerr.With(err, "min_expiration", s))
	}
	inputGet.MinExpiration = &d
	return nil
}

// LogLevel registers the flag for setting the logging level.
// Supported values are: debug, info, warn, error.
// Can be set via GHTKN_LOG_LEVEL environment variable.
func LogLevel(fs *pflag.FlagSet, dest *string) {
	fs.StringVar(dest, "log-level", "", "Log level (debug, info, warn, error)")
	cobrautil.Envs(fs, "log-level", env.LogLevel)
}

// Config registers the flag for specifying the configuration file path.
// Can be set via GHTKN_CONFIG environment variable.
// Alias: -c
func Config(fs *pflag.FlagSet, dest *string) {
	fs.StringVarP(dest, "config", "c", "", "configuration file path")
	cobrautil.Envs(fs, "config", env.Config)
}

// Format registers the flag for specifying the output format.
// Currently supports: json.
// Can be set via GHTKN_OUTPUT_FORMAT environment variable.
// Alias: -f
func Format(fs *pflag.FlagSet, dest *string) {
	fs.StringVarP(dest, "format", "f", "", "output format (json)")
	cobrautil.Envs(fs, "format", env.OutputFormat)
}

// MinExpiration registers the flag for specifying the minimum token expiration duration.
// Accepts duration strings like "1h", "30m", "30s".
// The GHTKN_MIN_EXPIRATION environment variable is read by the SDK, not this flag,
// so that it applies to SDK consumers too; the flag only carries the explicit -m value.
// Alias: -m
func MinExpiration(fs *pflag.FlagSet, dest *string) {
	fs.StringVarP(dest, "min-expiration", "m", "", "minimum expiration duration (e.g. 1h, 30m, 30s)")
}

// Clipboard registers the --clipboard (-p) flag, which copies the device flow one-time
// code to the system clipboard. The flag, when explicitly set, overrides both the
// GHTKN_CLIPBOARD environment variable and the config's clipboard.enable (both read
// by the SDK); it defaults to disabled.
// Alias: -p
func Clipboard(fs *pflag.FlagSet, dest *bool) {
	fs.BoolVarP(dest, "clipboard", "p", false, "Copy the device flow one-time code to the clipboard")
}
