// Package info implements the 'ghtkn info' command.
// It prints environment information useful for troubleshooting, such as the OS,
// architecture, ghtkn version, relevant environment variables (with tokens
// redacted), the selected backend, the target app, and the resolved
// configuration file path.
package info

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/flag"
	"github.com/suzuki-shunsuke/ghtkn/pkg/config"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/info"
	"github.com/suzuki-shunsuke/slog-util/slogutil"
	"github.com/suzuki-shunsuke/urfave-cli-v3-util/urfave"
	"github.com/urfave/cli/v3"
)

// Args holds the flag and argument values for the info command.
type Args struct {
	*flag.GlobalFlags

	AppName string // positional argument for 'info' command
	Version string
}

func New(logger *slogutil.Logger, env *urfave.Env, gFlags *flag.GlobalFlags) *cli.Command {
	args := &Args{
		GlobalFlags: gFlags,
		Version:     env.Version,
	}
	r := &runner{
		stdin: env.Stdin,
	}
	return r.Command(logger, args)
}

type runner struct {
	stdin io.Reader
}

func (r *runner) Command(logger *slogutil.Logger, args *Args) *cli.Command {
	return &cli.Command{
		Name:  "info",
		Usage: "Output information about the environment which is useful for troubleshooting",
		Action: func(ctx context.Context, _ *cli.Command) error {
			return r.action(ctx, logger, args)
		},
		Flags: []cli.Flag{
			flag.LogLevel(&args.LogLevel),
			flag.Config(&args.Config),
		},
		Arguments: []cli.Argument{
			&cli.StringArg{
				Name:        "app-name",
				Destination: &args.AppName,
			},
		},
	}
}

// action implements the 'info' command. It sets the log level, resolves the
// configuration file path, and delegates to the info controller to print the
// environment information. Returns an error if the config path can't be
// resolved or the controller fails.
func (r *runner) action(_ context.Context, logger *slogutil.Logger, args *Args) error {
	if err := logger.SetLevel(args.LogLevel); err != nil {
		return fmt.Errorf("set log level: %w", err)
	}
	// Resolve the config path here (honoring the -c flag, then the default)
	// so the controller stays free of environment lookups and is easy to test.
	configPath, err := config.ResolvePath(args.Config)
	if err != nil {
		return err //nolint:wrapcheck
	}
	return info.New(os.Stdout, os.Getenv).Info(configPath, args.AppName, args.Version) //nolint:wrapcheck
}
