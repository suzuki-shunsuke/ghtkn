// Package flag provides common command-line flags for ghtkn CLI.
// It defines reusable flag definitions for consistent flag handling across all commands.
package flag

import (
	"github.com/urfave/cli/v3"
)

// GlobalFlags holds the global flag values for the root command.
type GlobalFlags struct {
	LogLevel string
	Config   string
}

// LogLevel returns a flag for setting the logging level.
// Supported values are: debug, info, warn, error.
// Can be set via GHTKN_LOG_LEVEL environment variable.
func LogLevel(dest *string) *cli.StringFlag {
	return &cli.StringFlag{
		Name:        "log-level",
		Usage:       "Log level (debug, info, warn, error)",
		Sources:     cli.EnvVars("GHTKN_LOG_LEVEL"),
		Destination: dest,
	}
}

// Config returns a flag for specifying the configuration file path.
// Can be set via GHTKN_CONFIG environment variable.
// Alias: -c
func Config(dest *string) *cli.StringFlag {
	return &cli.StringFlag{
		Name:        "config",
		Aliases:     []string{"c"},
		Usage:       "configuration file path",
		Sources:     cli.EnvVars("GHTKN_CONFIG"),
		Destination: dest,
	}
}

// Format returns a flag for specifying the output format.
// Currently supports: json.
// Can be set via GHTKN_OUTPUT_FORMAT environment variable.
// Alias: -f
func Format(dest *string) *cli.StringFlag {
	return &cli.StringFlag{
		Name:        "format",
		Aliases:     []string{"f"},
		Usage:       "output format (json)",
		Sources:     cli.EnvVars("GHTKN_OUTPUT_FORMAT"),
		Destination: dest,
	}
}

// MinExpiration returns a flag for specifying the minimum token expiration duration.
// Accepts duration strings like "1h", "30m", "30s".
// The GHTKN_MIN_EXPIRATION environment variable is read by the SDK, not this flag,
// so that it applies to SDK consumers too; the flag only carries the explicit -m value.
// Alias: -m
func MinExpiration(dest *string) *cli.StringFlag {
	return &cli.StringFlag{
		Name:        "min-expiration",
		Aliases:     []string{"m"},
		Usage:       "minimum expiration duration (e.g. 1h, 30m, 30s)",
		Destination: dest,
	}
}

// DeviceFlow returns a flag controlling whether the OAuth device flow may run to
// create a new access token. It defaults to false, so the device flow is not
// started automatically; ghtkn fails fast with an actionable error instead of
// blocking in non-interactive environments. Pass -d (or -d=true) to allow it.
// The GHTKN_ENABLE_DEVICE_FLOW environment variable is read by the SDK, not this
// flag, so that it applies to SDK consumers too; the flag, when explicitly set,
// overrides the environment variable.
// Alias: -d
func DeviceFlow(dest *bool) *cli.BoolFlag {
	return &cli.BoolFlag{
		Name:        "device-flow",
		Aliases:     []string{"d"},
		Usage:       "Allow the interactive device flow to create a new access token",
		Value:       false,
		Destination: dest,
	}
}
