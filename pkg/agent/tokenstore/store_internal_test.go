package tokenstore

import "testing"

// TestValidClientID covers the file-name safety check directly; every other test in this
// package goes through the exported API and lives in store_test.go.
func TestValidClientID(t *testing.T) {
	t.Parallel()
	data := map[string]bool{
		"Iv1.abc": true,
		"Iv23xyz": true,
		"a_b-c.d": true,
		"":        false,
		".":       false,
		"..":      false,
		"a/b":     false,
		"a\x00b":  false,
		"a b":     false,
	}
	for id, want := range data {
		if got := validClientID(id); got != want {
			t.Errorf("validClientID(%q) = %v, want %v", id, got, want)
		}
	}
}
