package search

// ghtkn docs show ghtkn-backend
// ghtkn-backend/
//   SKILL.md
//   reference.md

import (
	"encoding/json"
	"io/fs"
	"strings"

	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/docs"
	"github.com/suzuki-shunsuke/ghtkn/skills"
)

func (c *Controller) Search(query string) error {
	results := []*docs.Result{}
	if err := fs.WalkDir(skills.FS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != "SKILL.md" {
			return nil
		}
		b, err := skills.FS.ReadFile(path)
		if err != nil {
			return err
		}
		result := &docs.Result{}
		if err := docs.Parse(b, result); err != nil {
			return err
		}
		if match(result, query) {
			results = append(results, result)
		}
		return nil
	}); err != nil {
		return err
	}
	encoder := json.NewEncoder(c.stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(results); err != nil {
		return err
	}
	return nil
}

// match returns true if the result matches the query.
// If result.Description contains the query, returns true.
func match(result *docs.Result, query string) bool {
	if result == nil {
		return false
	}
	return strings.Contains(result.Description, query)
}
