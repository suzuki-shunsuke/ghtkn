package docs

import (
	"github.com/spf13/cobra"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/docs/list"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/docs/show"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/flag"
	"github.com/suzuki-shunsuke/slog-util/slogutil"
)

// Hint points coding agents at `ghtkn docs`. It lives here, next to the command it
// advertises, because several commands emit it: an agent that never makes ghtkn fail
// only learns that these documents exist if a command it does run says so.
//
// Callers log it to stderr rather than writing it to stdout, so that it doesn't break
// scripts that parse the output, and at the info level so that `--log-level warn`
// silences it.
const Hint = "If you are a coding agent, run `ghtkn docs list` to list the documentation and `ghtkn docs show <name>` to read it before answering questions about ghtkn or troubleshooting its errors."

// Args holds the flag and argument values for the info command.
type Args struct {
	*flag.GlobalFlags
}

func New(logger *slogutil.Logger, gFlags *flag.GlobalFlags) *cobra.Command {
	args := &Args{
		GlobalFlags: gFlags,
	}
	r := &runner{}
	return r.Command(logger, args)
}

type runner struct{}

func (r *runner) Command(logger *slogutil.Logger, args *Args) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "docs",
		Short: "Output document for coding agent",
		Long:  `List and output documents. This command is useful for coding agent to read document and solve problems.`,
	}
	cmd.AddCommand(
		show.New(logger, args.GlobalFlags),
		list.New(logger, args.GlobalFlags),
	)
	return cmd
}
