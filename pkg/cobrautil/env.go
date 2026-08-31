package cobrautil

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// envAnnotation is the pflag annotation holding the environment variables a flag
// falls back to. pflag has no notion of environment variables, so the names are
// carried on the flag itself and read back by ApplyEnvs.
const envAnnotation = "cobrautil_envs"

// Envs makes the named flag fall back to the given environment variables, in order,
// when the flag is not given on the command line. It is the counterpart of
// urfave/cli's cli.EnvVars.
//
// The names are appended to the flag's usage, as "[$FOO]", so that the help says
// where else the value can come from, the way urfave/cli's help did.
//
// It panics when the flag is not registered, since that is a mistake in the command
// definition rather than something a user can cause.
func Envs(fs *pflag.FlagSet, name string, envs ...string) {
	f := fs.Lookup(name)
	if f == nil {
		panic("cobrautil: no such flag: " + name)
	}
	if len(envs) == 0 {
		return
	}
	if err := fs.SetAnnotation(name, envAnnotation, envs); err != nil {
		// SetAnnotation fails only when the flag is not registered, which the lookup
		// above has already ruled out.
		panic("cobrautil: set the environment variable annotation: " + err.Error())
	}
	for _, env := range envs {
		f.Usage += " [$" + env + "]"
	}
}

// ApplyEnvs sets, from the environment, every flag of cmd that has environment
// variables bound by Envs and was not given on the command line. The first variable
// with a non-empty value wins, so the precedence is the command line, then the
// environment variables in the order they were bound, then the flag's default.
//
// An empty value is treated as unset, unlike urfave/cli, which used os.LookupEnv and
// so let an empty value override the default. The two differ only for a variable set
// to the empty string, where the value applied is the empty string either way for
// every flag type we use.
//
// Command wires this into the root command's PersistentPreRunE, so commands don't
// call it themselves. cmd is the command being run, whose flag set holds the
// persistent flags it inherits as well as its own.
func ApplyEnvs(cmd *cobra.Command, getenv func(string) string) error {
	if getenv == nil {
		return nil
	}
	var err error
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if err != nil || f.Changed {
			return
		}
		for _, env := range f.Annotations[envAnnotation] {
			v := getenv(env)
			if v == "" {
				continue
			}
			if e := cmd.Flags().Set(f.Name, v); e != nil {
				err = fmt.Errorf("set the flag --%s from the environment variable %s: %w", f.Name, env, e)
			}
			return
		}
	})
	return err
}
