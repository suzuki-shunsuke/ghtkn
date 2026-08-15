// Package jsonschema implements the 'ghtkn json-schema' command.
// It outputs the JSON Schema of the configuration file, which is embedded in the
// binary, so that editors can complete and validate the configuration file.
package jsonschema

import (
	"fmt"

	"github.com/spf13/cobra"
	schema "github.com/suzuki-shunsuke/ghtkn/json-schema"
)

// New builds the 'json-schema' command. It takes neither the logger nor the global
// flags, unlike the other commands, because it only writes an embedded file: it reads
// no configuration and logs nothing.
func New() *cobra.Command {
	return &cobra.Command{
		Use:   "json-schema",
		Short: "Output JSON Schema for the configuration file",
		Long: `Output JSON Schema for the configuration file.

The schema is embedded in the binary, so it always describes the configuration
this version of ghtkn accepts. Editors such as VSCode use it to complete the
configuration file and to warn about invalid settings.

$ ghtkn json-schema > ghtkn.json`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			// The schema ends with a newline, so nothing is added to it. OutOrStdout
			// rather than os.Stdout, so that a test can capture the output.
			fmt.Fprint(cmd.OutOrStdout(), string(schema.Schema))
		},
	}
}
