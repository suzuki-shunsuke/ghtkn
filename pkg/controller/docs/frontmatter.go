package docs

import (
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

type Result struct {
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description" yaml:"description"`
}

// Parse parses the frontmatter of a SKILL.md file and returns a Result.
func Parse(b []byte, result *Result) error {
	content := string(b)
	const delim = "---\n"
	if !strings.HasPrefix(content, delim) {
		return errors.New("SKILL.md has no frontmatter")
	}
	front, _, found := strings.Cut(content[len(delim):], "\n---")
	if !found {
		return errors.New("SKILL.md frontmatter is not closed")
	}
	if err := yaml.Unmarshal([]byte(front), result); err != nil {
		return fmt.Errorf("parse frontmatter as YAML: %w", err)
	}
	return nil
}
