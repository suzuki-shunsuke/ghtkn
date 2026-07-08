package show

import (
	"context"
	"fmt"
	"os"

	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/flag"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/docs/show"
	"github.com/suzuki-shunsuke/slog-util/slogutil"
	"github.com/urfave/cli/v3"
)

// Args holds the flag and argument values for the info command.
type Args struct {
	*flag.GlobalFlags

	DocName string
}

func New(logger *slogutil.Logger, gFlags *flag.GlobalFlags) *cli.Command {
	args := &Args{
		GlobalFlags: gFlags,
	}
	r := &runner{}
	return r.Command(logger, args)
}

type runner struct{}

func (r *runner) Command(logger *slogutil.Logger, args *Args) *cli.Command {
	return &cli.Command{
		Name:  "show",
		Usage: "Output the content of a given document",
		Description: `Output document. This is useful for coding agent to read the document and solve problems.
This command needs a document name.
To see the name, list documents with "ghtkn docs list"`,
		Action: func(ctx context.Context, _ *cli.Command) error {
			return r.action(ctx, logger, args)
		},
		Flags: []cli.Flag{
			flag.LogLevel(&args.LogLevel),
		},
		Arguments: []cli.Argument{
			&cli.StringArg{
				Name:        "doc",
				Destination: &args.DocName,
			},
		},
	}
}

func (r *runner) action(_ context.Context, logger *slogutil.Logger, args *Args) error {
	if err := logger.SetLevel(args.LogLevel); err != nil {
		return fmt.Errorf("set log level: %w", err)
	}
	return show.New(os.Stdout).Show(args.DocName) //nolint:wrapcheck
}
