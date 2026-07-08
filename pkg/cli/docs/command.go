package docs

import (
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/docs/list"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/docs/show"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/flag"
	"github.com/suzuki-shunsuke/slog-util/slogutil"
	"github.com/urfave/cli/v3"
)

// Args holds the flag and argument values for the info command.
type Args struct {
	*flag.GlobalFlags
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
		Name:        "docs",
		Usage:       "Output document for coding agent",
		Description: `List and output documents. This command is useful for coding agent to read document and solve problems.`,
		Commands: []*cli.Command{
			show.New(logger, args.GlobalFlags),
			list.New(logger, args.GlobalFlags),
		},
	}
}
