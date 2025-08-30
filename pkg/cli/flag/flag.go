package flag

import (
	"github.com/urfave/cli/v3"
)

func LogLevel() *cli.StringFlag {
	return &cli.StringFlag{
		Name:    "log-level",
		Usage:   "Log level (debug, info, warn, error)",
		Sources: cli.EnvVars("GHTKN_LOG_LEVEL"),
	}
}

func LogLevelValue(c *cli.Command) string {
	return c.String("log-level")
}

func Config() *cli.StringFlag {
	return &cli.StringFlag{
		Name:    "config",
		Aliases: []string{"c"},
		Usage:   "configuration file path",
		Sources: cli.EnvVars("GHTKN_CONFIG"),
	}
}

func ConfigValue(c *cli.Command) string {
	return c.String("config")
}

func Format() *cli.StringFlag {
	return &cli.StringFlag{
		Name:    "format",
		Aliases: []string{"f"},
		Usage:   "output format (json)",
		Sources: cli.EnvVars("GHTKN_OUTPUT_FORMAT"),
	}
}

func FormatValue(c *cli.Command) string {
	return c.String("format")
}

func MinExpiration() *cli.StringFlag {
	return &cli.StringFlag{
		Name:    "min-expiration",
		Aliases: []string{"m"},
		Usage:   "minimum expiration duration (e.g. 1h, 30m, 30s)",
	}
}

func MinExpirationValue(c *cli.Command) string {
	return c.String("min-expiration")
}
