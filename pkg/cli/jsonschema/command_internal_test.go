package jsonschema

import (
	"bytes"
	"encoding/json"
	"testing"
)

// TestNew checks that the command writes the schema and nothing else to stdout, so
// that `ghtkn json-schema > ghtkn.json` yields a file editors can read.
func TestNew(t *testing.T) {
	t.Parallel()
	cmd := New()
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs(nil)

	if err := cmd.ExecuteContext(t.Context()); err != nil {
		t.Fatal(err)
	}

	m := map[string]any{}
	if err := json.Unmarshal(stdout.Bytes(), &m); err != nil {
		t.Fatalf("the output isn't a JSON object: %v: %s", err, stdout.String())
	}
	if _, ok := m["$schema"]; !ok {
		t.Errorf("the output isn't a JSON Schema; it has no $schema: %s", stdout.String())
	}
}
