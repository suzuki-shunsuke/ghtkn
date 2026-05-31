package agent

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/term"
)

// errPassphraseMismatch is returned when the two entries during first-run
// passphrase creation do not match.
var errPassphraseMismatch = errors.New("passphrases do not match")

// readPassphrase reads a passphrase from the controlling terminal without echoing
// it. The prompt is written to stderr. It returns an error when stdin is not a
// terminal: the passphrase is never read from an environment variable or a pipe.
func readPassphrase(prompt string) ([]byte, error) {
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

// promptPassphrase prompts for the agent passphrase. When the key file does not yet
// exist (exists == false) it prompts twice and verifies the entries match, because
// the passphrase is the only way to ever decrypt tokens written under it.
func (c *Controller) promptPassphrase(exists bool) ([]byte, error) {
	if exists {
		return c.readPassphrase("Enter the agent passphrase: ")
	}
	pass, err := c.readPassphrase("Enter a new agent passphrase: ")
	if err != nil {
		return nil, err
	}
	confirm, err := c.readPassphrase("Confirm the agent passphrase: ")
	if err != nil {
		return nil, err
	}
	if string(pass) != string(confirm) {
		return nil, errPassphraseMismatch
	}
	return pass, nil
}
