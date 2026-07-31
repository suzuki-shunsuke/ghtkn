package docs

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"gopkg.in/yaml.v3"
)

type Result struct {
	// Name is the document name. It comes from the file name rather than the
	// frontmatter, so the two can't drift apart.
	Name        string `json:"name" yaml:"-"`
	Description string `json:"description" yaml:"description"`
}

// Ext is the file extension of documents.
const Ext = ".md"

// Names lists the document names in fsys.
func Names(fsys fs.FS) ([]string, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, fmt.Errorf("read the docs directory: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), Ext) {
			continue
		}
		names = append(names, strings.TrimSuffix(entry.Name(), Ext))
	}
	return names, nil
}

// Parse parses the frontmatter of a document and sets its description to a Result.
func Parse(b []byte, result *Result) error {
	content := string(b)
	const delim = "---\n"
	if !strings.HasPrefix(content, delim) {
		return errors.New("the document has no frontmatter")
	}
	front, _, found := strings.Cut(content[len(delim):], "\n---")
	if !found {
		return errors.New("the document frontmatter is not closed")
	}
	if err := yaml.Unmarshal([]byte(front), result); err != nil {
		return fmt.Errorf("parse frontmatter as YAML: %w", err)
	}
	return nil
}
