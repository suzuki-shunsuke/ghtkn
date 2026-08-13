package cobrautil

import (
	"github.com/spf13/cobra"
)

// Options configures the pieces Command adds to the root command.
type Options struct {
	// SHA is the commit the binary was built from, reported by 'version --json'.
	SHA string
	// AfterVersion, when set, is called after the version has been printed, by
	// --version, -v, and the version command alike. It is the hook for output that
	// should accompany the version wherever it is asked for.
	//
	// urfave/cli offered the process global cli.VersionPrinter for this, which had to
	// be saved and restored around every test that touched it; a field on the root
	// command has no such reach.
	AfterVersion func()
}

// Command wires the root command to the process environment and adds the commands
// and flags every CLI of ours has: --version/-v, the version command, and the hidden
// help-all command.
//
// Errors and usage are silenced so that Execute returns the error to the caller
// instead of printing it, leaving the reporting and the exit code to Main.
//
// The environment variables bound with Envs are applied by the root's
// PersistentPreRunE, so a command that sets one of its own must call ApplyEnvs
// itself: cobra runs only the closest PersistentPreRunE in the chain.
func Command(env *Env, cmd *cobra.Command, opts *Options) *cobra.Command {
	if opts == nil {
		opts = &Options{}
	}
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	if env.Args != nil {
		// The program name is not an argument to cobra, unlike urfave/cli.
		cmd.SetArgs(env.Args[1:])
	}
	if env.Stdin != nil {
		cmd.SetIn(env.Stdin)
	}
	if env.Stdout != nil {
		cmd.SetOut(env.Stdout)
	}
	if env.Stderr != nil {
		cmd.SetErr(env.Stderr)
	}
	cmd.PersistentPreRunE = func(c *cobra.Command, _ []string) error {
		return ApplyEnvs(c, env.Getenv)
	}
	setVersionFlag(cmd, env, opts)
	cmd.AddCommand(versionCommand(env, opts), helpAllCommand())
	return cmd
}

// setVersionFlag registers --version/-v and handles it in the root command's RunE.
//
// cobra prints the version itself when the root command's Version is set, but it does
// so before the PersistentPreRunE hooks and returns without reaching RunE, which
// leaves nowhere to hook AfterVersion in. Leaving Version empty and owning the flag
// here puts the printing after the environment variables have been applied, so
// AfterVersion sees the same flag values a subcommand would.
func setVersionFlag(cmd *cobra.Command, env *Env, opts *Options) {
	version := false
	cmd.Flags().BoolVarP(&version, "version", "v", false, "print the version")
	runE := cmd.RunE
	cmd.RunE = func(c *cobra.Command, args []string) error {
		if version {
			printVersion(c, env, opts)
			return nil
		}
		if runE != nil {
			return runE(c, args)
		}
		// A root command with no action of its own shows its help, as urfave/cli did.
		return c.Help()
	}
}
