// Package get implements both the 'ghtkn get' command and 'ghtkn git-credential' command.
// These commands retrieve or create GitHub App User Access Tokens and output them to stdout.
// The 'get' command outputs tokens in plain text or JSON format for general use.
// The 'git-credential' command outputs tokens in Git's credential helper format for seamless Git authentication.
// Both commands handle token persistence, expiration checking, and automatic renewal when needed.
package get

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/flag"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/get"
	"github.com/suzuki-shunsuke/ghtkn/pkg/log"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
	"github.com/urfave/cli/v3"
)

// New creates either a 'get' or 'git-credential' command instance based on the isGitCredential flag.
// When isGitCredential is true, it creates a Git credential helper command.
// When false, it creates a standard get command for general token retrieval.
// It returns a CLI command that can be registered with the main CLI application.
func New(logger *slog.Logger, version string, isGitCredential bool) *cli.Command {
	r := &runner{
		logger:          logger,
		version:         version,
		isGitCredential: isGitCredential,
		stdin:           os.Stdin,
	}
	return r.Command()
}

// runner encapsulates the state and behavior for both the get and git-credential commands.
type runner struct {
	logger          *slog.Logger
	version         string
	isGitCredential bool
	stdin           io.Reader
}

// Command returns the CLI command definition for either the get or git-credential subcommand.
// For git-credential, it creates a command compatible with Git's credential helper protocol.
// For get, it creates a standard command with output format options.
// It defines the command name, usage, action handler, and available flags.
func (r *runner) Command() *cli.Command {
	if r.isGitCredential {
		return &cli.Command{
			Name:   "git-credential",
			Usage:  "Git Credential Helper",
			Action: r.action,
			Flags: []cli.Flag{
				flag.LogLevel(),
				flag.Config(),
				flag.MinExpiration(),
			},
		}
	}
	return &cli.Command{
		Name:   "get",
		Usage:  "Output a GitHub App User Access Token to stdout",
		Action: r.action,
		Flags: []cli.Flag{
			flag.LogLevel(),
			flag.Config(),
			flag.Format(),
			flag.MinExpiration(),
		},
	}
}

// action implements the main logic for both the get and git-credential commands.
// For git-credential, it follows Git's credential helper protocol and only processes 'get' operations.
// For get command, it supports different output formats (plain text or JSON).
// It configures the controller with flags and arguments, then executes the token retrieval.
// Returns an error if configuration is invalid or token retrieval fails.
func (r *runner) action(ctx context.Context, c *cli.Command) error { //nolint:cyclop
	inputGet := &ghtkn.InputGet{}
	if m := flag.MinExpirationValue(c); m != "" {
		d, err := time.ParseDuration(m)
		if err != nil {
			return fmt.Errorf("parse the min expiration: %w", slogerr.With(err, "min_expiration", m))
		}
		inputGet.MinExpiration = d
	}
	inputGet.ConfigFilePath = flag.ConfigValue(c)

	input := get.NewInput()
	if r.isGitCredential {
		input.IsGitCredential = true
		if arg := c.Args().First(); arg != "get" {
			return nil
		}
		if _, err := r.readStdinForGitCredentialHelper(ctx); err != nil {
			return fmt.Errorf("read stdin: %w", err)
		}
	} else {
		input.OutputFormat = flag.FormatValue(c)
		if arg := c.Args().First(); arg != "" {
			inputGet.AppName = arg
		}
	}
	logger := r.logger
	if lvlS := flag.LogLevelValue(c); lvlS != "" {
		lvl, err := log.ParseLevel(lvlS)
		if err != nil {
			return fmt.Errorf("parse the log level: %w", slogerr.With(err, "log_level", lvlS))
		}
		logger = log.New(r.version, lvl)
	}
	if inputGet.ConfigFilePath == "" {
		p, err := ghtkn.GetConfigPath()
		if err != nil {
			return fmt.Errorf("get the config path: %w", err)
		}
		inputGet.ConfigFilePath = p
	}
	if err := input.Validate(); err != nil {
		return err //nolint:wrapcheck
	}
	return get.New(input).Run(ctx, logger, inputGet) //nolint:wrapcheck
}
