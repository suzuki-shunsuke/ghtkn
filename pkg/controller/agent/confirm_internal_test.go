package agent

import (
	"strings"
	"testing"
)

func TestReadConfirm(t *testing.T) {
	t.Parallel()
	data := map[string]bool{
		"y\n":      true,
		"Y\n":      true,
		"yes\n":    true,
		"YES\n":    true,
		"\tyes \n": true, // surrounding whitespace is trimmed
		"y":        true, // no trailing newline
		"n\n":      false,
		"no\n":     false,
		"\n":       false,
		"":         false,
		"yep\n":    false,
	}
	for input, want := range data {
		got, err := readConfirm(strings.NewReader(input))
		if err != nil {
			t.Fatalf("readConfirm(%q) error: %v", input, err)
		}
		if got != want {
			t.Errorf("readConfirm(%q) = %v, want %v", input, got, want)
		}
	}
}
