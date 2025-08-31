package get

import (
	"encoding/json"
	"fmt"

	"github.com/suzuki-shunsuke/ghtkn/pkg/keyring"
)

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

func (c *Controller) outputJSON(data any) error {
	encoder := json.NewEncoder(c.input.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("encode as JSON: %w", err)
	}
	return nil
}
