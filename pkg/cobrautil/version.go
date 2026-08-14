package cobrautil

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

// versionCommand returns the 'version' command, which prints the version on its own
// line, or as JSON with --json, so that it can be read by a script as well as by a
// person.
func versionCommand(env *Env, opts *Options) *cobra.Command {
	asJSON := false
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			if asJSON {
				if err := printVersionJSON(c, env, opts); err != nil {
					return err
				}
			} else {
				fmt.Fprintln(c.OutOrStdout(), versionOrUnknown(env.Version))
			}
			if opts.AfterVersion != nil {
				opts.AfterVersion()
			}
			return nil
		},
	}
	cmd.Flags().BoolVarP(&asJSON, "json", "j", false, "Output version in JSON format")
	return cmd
}

// printVersion writes the line --version and -v print, which names the program so
// that it stays readable when several versions are pasted together.
func printVersion(cmd *cobra.Command, env *Env, opts *Options) {
	fmt.Fprintf(cmd.OutOrStdout(), "%s version %s\n", env.Program, versionOrUnknown(env.Version))
	if opts.AfterVersion != nil {
		opts.AfterVersion()
	}
}

func printVersionJSON(cmd *cobra.Command, env *Env, opts *Options) error {
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(map[string]string{
		"name":    env.Program,
		"version": env.Version,
		"sha":     opts.SHA,
	}); err != nil {
		return fmt.Errorf("encode JSON: %w", err)
	}
	return nil
}

// versionOrUnknown keeps the output a single, parsable word for a binary built
// without version information, such as one from 'go install'.
func versionOrUnknown(version string) string {
	if version == "" {
		return "unknown"
	}
	return version
}
