package get

import (
	"encoding/json"
	"fmt"

	"github.com/suzuki-shunsuke/ghtkn/pkg/keyring"
)

// output writes the access token to stdout in the configured format.
// It outputs either the raw token string (default) or a JSON object based on OutputFormat.
func (c *Controller) output(token *keyring.AccessToken) error {
	// Output access token
	if c.input.IsJSON() {
		// JSON format
		if err := c.outputJSON(token); err != nil {
			return fmt.Errorf("output access token: %w", err)
		}
		return nil
	}
	fmt.Fprintln(c.input.Stdout, token.AccessToken)
	return nil
}

// outputJSON encodes the given data as formatted JSON and writes it to stdout.
// The JSON is indented with two spaces for readability.
func (c *Controller) outputJSON(data any) error {
	encoder := json.NewEncoder(c.input.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("encode as JSON: %w", err)
	}
	return nil
}
