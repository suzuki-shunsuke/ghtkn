package tty

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// Confirm asks the user a yes/no question on the controlling terminal and reports
// whether they answered yes. The prompt is written to stderr. It returns an error
// when stdin is not a terminal, so a destructive operation is never confirmed
// non-interactively (e.g. from a pipe).
func Confirm(prompt string) (bool, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return false, errors.New("a terminal is required to confirm this operation")
	}
	fmt.Fprint(os.Stderr, prompt)
	return readConfirm(os.Stdin)
}

// readConfirm reads one line from r and reports whether it is an affirmative
// answer ("y" or "yes", case-insensitive). It is split out so it can be tested
// without a terminal.
func readConfirm(r io.Reader) (bool, error) {
	line, err := bufio.NewReader(r).ReadString('\n')
	// ReadString returns io.EOF together with the data when there is no trailing
	// newline, so a non-empty answer without a newline is still valid.
	if err != nil && !errors.Is(err, io.EOF) {
		return false, fmt.Errorf("read the confirmation: %w", err)
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}
