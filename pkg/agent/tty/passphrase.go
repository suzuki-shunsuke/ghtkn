// Package tty provides terminal helpers shared by the agent subcommands: reading a
// passphrase without echo and asking a yes/no confirmation. They all require a real
// terminal and never read from an environment variable or a pipe, so the agent
// passphrase and destructive confirmations stay interactive-only.
package tty

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/term"
)

// ErrPassphraseMismatch is returned when the two entries during first-run
// passphrase creation do not match.
var ErrPassphraseMismatch = errors.New("passphrases do not match")

// ReadPassphrase reads a passphrase from the controlling terminal without echoing
// it. The prompt is written to stderr. It returns an error when stdin is not a
// terminal: the passphrase is never read from an environment variable or a pipe.
func ReadPassphrase(prompt string) ([]byte, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return nil, errors.New("a terminal is required to enter the agent passphrase")
	}
	fmt.Fprint(os.Stderr, prompt)
	pass, err := term.ReadPassword(fd)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("read the passphrase: %w", err)
	}
	return pass, nil
}

// PromptPassphrase prompts for the agent passphrase using read. When the key file
// does not yet exist (exists == false) it prompts twice and verifies the entries
// match, because the passphrase is the only way to ever decrypt tokens written
// under it. read is injected so callers (and tests) can supply their own reader.
func PromptPassphrase(read func(prompt string) ([]byte, error), exists bool) ([]byte, error) {
	if exists {
		return read("Enter the agent passphrase: ")
	}
	pass, err := read("Enter a new agent passphrase: ")
	if err != nil {
		return nil, err
	}
	confirm, err := read("Confirm the agent passphrase: ")
	if err != nil {
		return nil, err
	}
	if string(pass) != string(confirm) {
		return nil, ErrPassphraseMismatch
	}
	return pass, nil
}
