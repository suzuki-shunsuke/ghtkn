// Package initcmd implements the 'ghtkn init' command.
// This package is responsible for generating ghtkn configuration files (.ghtkn.yaml)
// with default settings to help users quickly set up ghtkn in their repositories.
// It creates configuration templates that define target workflow files and
// action ignore patterns for the pinning process.
package initcmd

import (
	"context"
	"fmt"

	"github.com/spf13/afero"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/flag"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/initcmd"
	"github.com/suzuki-shunsuke/slog-util/slogutil"
	"github.com/urfave/cli/v3"
)

// Args holds the flag and argument values for the init command.
type Args struct {
	*flag.GlobalFlags

	ConfigFilePath string // positional argument
}

// New creates a new init command instance with the provided logger.
// It returns a CLI command that can be registered with the main CLI application.
func New(logger *slogutil.Logger, gFlags *flag.GlobalFlags) *cli.Command {
	args := &Args{
		GlobalFlags: gFlags,
	}
	return &cli.Command{
		Name:  "init",
		Usage: "Create ghtkn.yaml if it doesn't exist",
		Action: func(ctx context.Context, _ *cli.Command) error {
			return action(ctx, logger, args)
		},
		Flags: []cli.Flag{
			flag.LogLevel(&args.LogLevel),
			flag.Config(&args.Config),
		},
		Arguments: []cli.Argument{
			&cli.StringArg{
				Name:        "config-file-path",
				Destination: &args.ConfigFilePath,
			},
		},
	}
}

func action(_ context.Context, logger *slogutil.Logger, args *Args) error {
	if err := logger.SetLevel(args.LogLevel); err != nil {
		return fmt.Errorf("set log level: %w", err)
	}

	configFilePath := args.ConfigFilePath
	if configFilePath == "" {
		configFilePath = args.Config
	}
	if configFilePath == "" {
		p, err := ghtkn.GetConfigPath()
		if err != nil {
			return fmt.Errorf("get the config path: %w", err)
		}
		configFilePath = p
	}
	fs := afero.NewOsFs()
	ctrl := initcmd.New(fs)
	return ctrl.Init(logger.Logger, configFilePath) //nolint:wrapcheck
}
