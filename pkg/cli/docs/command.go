// Package docs adds the docs command to ghtkn and holds the hint that points coding
// agents at it.
//
// The command itself is cobra-util's, which lists the documents embedded in the
// binary and outputs them; ghtkn only says which documents those are and what the
// program is called.
package docs

import (
	"github.com/spf13/cobra"
	cobradocs "github.com/suzuki-shunsuke/cobra-util/docs"
	docsfs "github.com/suzuki-shunsuke/ghtkn/docs"
)

// command describes the documents ghtkn embeds. It is built per call rather than
// kept in a package variable so that New and Hint say the same thing without sharing
// state.
func command() *cobradocs.Command {
	return &cobradocs.Command{
		FS:   docsfs.FS,
		Name: "ghtkn",
	}
}

// New returns the `ghtkn docs` command and its `list` and `show` subcommands.
//
// Unlike the other commands it takes neither the logger nor the global flags: it
// reads no configuration and logs nothing.
func New() *cobra.Command {
	return cobradocs.New(command())
}

// Hint points coding agents at `ghtkn docs`. It lives here, next to the command it
// advertises, because several commands emit it: an agent that never makes ghtkn fail
// only learns that these documents exist if a command it does run says so.
//
// Callers log it to stderr rather than writing it to stdout, so that it doesn't break
// scripts that parse the output, and at the info level so that `--log-level warn`
// silences it.
func Hint() string {
	return command().Hint()
}
