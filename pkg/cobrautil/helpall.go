package cobrautil

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
)

// helpAllCommandName is the name of the command, which is also the name it skips
// while walking the tree so that it doesn't document itself.
const helpAllCommandName = "help-all"

// helpAllCommand returns the hidden 'help-all' command, which writes the help of
// every command as Markdown. It is what generates the USAGE.md of our CLIs, so its
// output is a document rather than something meant to be read in a terminal.
func helpAllCommand() *cobra.Command {
	return &cobra.Command{
		Use:    helpAllCommandName,
		Short:  "show all help",
		Args:   cobra.NoArgs,
		Hidden: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			root := cmd.Root()
			w := cmd.OutOrStdout()
			fmt.Fprintln(w, "```console")
			fmt.Fprintf(w, "$ %s --help\n", root.Name())
			// cobra adds --help to a command when it runs it, and the command running
			// here is help-all; without this the documented help is the only place
			// --help is missing from.
			root.InitDefaultHelpFlag()
			if err := root.Help(); err != nil {
				return fmt.Errorf("show the help of %s: %w", root.Name(), err)
			}
			fmt.Fprintln(w, "```")
			for _, c := range root.Commands() {
				if err := showCommandHelp(w, c, 2); err != nil { //nolint:mnd // The root's own help is a level above, and it is the document's title.
					return err
				}
			}
			return nil
		},
	}
}

// showCommandHelp writes the help of cmd and of its subcommands, each under a
// Markdown heading of the given level.
//
// The commands that exist to serve the terminal rather than to be documented are
// skipped: the hidden ones, help-all itself, and cobra's 'help', whose help only
// explains how to ask for help.
func showCommandHelp(w io.Writer, cmd *cobra.Command, level int) error {
	if cmd.Hidden || cmd.Name() == "help" || cmd.Name() == helpAllCommandName {
		return nil
	}
	path := cmd.CommandPath()
	fmt.Fprintf(w, "\n%s %s\n\n", strings.Repeat("#", level), path)
	fmt.Fprintln(w, "```console")
	fmt.Fprintf(w, "$ %s --help\n", path)
	cmd.InitDefaultHelpFlag()
	if err := cmd.Help(); err != nil {
		return fmt.Errorf("show the help of %s: %w", path, err)
	}
	fmt.Fprintln(w, "```")
	for _, sub := range cmd.Commands() {
		if err := showCommandHelp(w, sub, level+1); err != nil {
			return err
		}
	}
	return nil
}
