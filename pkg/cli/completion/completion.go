// Package completion implements shell completion of the app name arguments that
// 'ghtkn get', 'ghtkn auth', and 'ghtkn revoke' take. The names come from the
// configuration file, so the shell offers exactly the apps that ghtkn can issue a
// token for.
//
// Shell completion itself comes from cobra's 'completion' command; this package only
// supplies the candidates. Unlike urfave/cli, cobra completes flags before it asks
// for argument candidates, so nothing here has to hand that case back.
package completion

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn"
)

// Func is the signature cobra takes for completing positional arguments.
type Func func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective)

// AppName returns a completion function for a command that takes a single app name,
// such as 'ghtkn get'. configFilePath points at the -c flag's destination, which is
// read when the completion runs rather than when the command is built.
func AppName(configFilePath *string) Func {
	return func(_ *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 {
			// The app name is already given, so there is nothing left to complete.
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return appNames(*configFilePath, toComplete, nil), cobra.ShellCompDirectiveNoFileComp
	}
}

// AppNames is AppName for a command that takes any number of app names, namely
// 'ghtkn revoke'. Arguments already on the command line are dropped from the
// candidates, so an app is not offered twice.
//
// ignored reports whether the command line being completed makes the command ignore
// its app name arguments; no app is offered then. It must not be nil.
func AppNames(configFilePath *string, ignored func(cmd *cobra.Command) bool) Func {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if ignored(cmd) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return appNames(*configFilePath, toComplete, args), cobra.ShellCompDirectiveNoFileComp
	}
}

// appNames returns the name of every configured app that starts with toComplete,
// except those in exclude.
//
// Errors are dropped: the output is the candidate list itself, so a message about a
// broken config file would be offered as a candidate. Nothing here reaches the
// keyring, the agent, or GitHub either, because this runs on every press of the tab
// key.
func appNames(configFilePath, toComplete string, exclude []string) []string {
	// LoadConfig resolves GHTKN_CONFIG itself when the path is empty, which is what
	// makes the environment variable work here: the -c flag's own environment variable
	// source is applied by the root command's PersistentPreRunE, which cobra does not
	// run while completing. A missing config file is not an error, and yields no
	// candidate.
	cfg, err := ghtkn.LoadConfig(&ghtkn.InputLoadConfig{ConfigFilePath: configFilePath})
	if err != nil {
		return nil
	}
	// A name is offered once. The config file can hold duplicates: Config.Validate
	// rejects them, but nothing validates it here, and a candidate listed twice is
	// noise rather than a report of the problem.
	seen := make(map[string]struct{}, len(exclude))
	for _, name := range exclude {
		seen[name] = struct{}{}
	}
	var names []string
	for _, app := range cfg.Apps {
		if app == nil || app.Name == "" {
			continue
		}
		if _, ok := seen[app.Name]; ok {
			continue
		}
		seen[app.Name] = struct{}{}
		if !strings.HasPrefix(app.Name, toComplete) {
			continue
		}
		names = append(names, app.Name)
	}
	return names
}
