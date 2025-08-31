// Package flag provides common command-line flags for ghtkn CLI.
// It defines reusable flag definitions and value accessors for consistent
// flag handling across all commands.
package flag

import (
	"github.com/urfave/cli/v3"
)

// LogLevel returns a flag for setting the logging level.
// Supported values are: debug, info, warn, error.
// Can be set via GHTKN_LOG_LEVEL environment variable.
func LogLevel() *cli.StringFlag {
	return &cli.StringFlag{
		Name:    "log-level",
		Usage:   "Log level (debug, info, warn, error)",
		Sources: cli.EnvVars("GHTKN_LOG_LEVEL"),
	}
}

// LogLevelValue retrieves the log level value from the command context.
func LogLevelValue(c *cli.Command) string {
	return c.String("log-level")
}

// Config returns a flag for specifying the configuration file path.
// Can be set via GHTKN_CONFIG environment variable.
// Alias: -c
func Config() *cli.StringFlag {
	return &cli.StringFlag{
		Name:    "config",
		Aliases: []string{"c"},
		Usage:   "configuration file path",
		Sources: cli.EnvVars("GHTKN_CONFIG"),
	}
}

// ConfigValue retrieves the config file path from the command context.
func ConfigValue(c *cli.Command) string {
	return c.String("config")
}

// Format returns a flag for specifying the output format.
// Currently supports: json.
// Can be set via GHTKN_OUTPUT_FORMAT environment variable.
// Alias: -f
func Format() *cli.StringFlag {
	return &cli.StringFlag{
		Name:    "format",
		Aliases: []string{"f"},
		Usage:   "output format (json)",
		Sources: cli.EnvVars("GHTKN_OUTPUT_FORMAT"),
	}
}

// FormatValue retrieves the output format value from the command context.
func FormatValue(c *cli.Command) string {
	return c.String("format")
}

// MinExpiration returns a flag for specifying the minimum token expiration duration.
// Accepts duration strings like "1h", "30m", "30s".
// Alias: -m
func MinExpiration() *cli.StringFlag {
	return &cli.StringFlag{
		Name:    "min-expiration",
		Aliases: []string{"m"},
		Usage:   "minimum expiration duration (e.g. 1h, 30m, 30s)",
		Sources: cli.EnvVars("GHTKN_MIN_EXPIRATION"),
	}
}

// MinExpirationValue retrieves the minimum expiration duration from the command context.
func MinExpirationValue(c *cli.Command) string {
	return c.String("min-expiration")
}
