package list

import (
	"encoding/json"
	"fmt"
	"io/fs"

	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/docs"
	"github.com/suzuki-shunsuke/ghtkn/skills"
)

type Results struct {
	Results []*docs.Result `json:"results"`
	Help    string         `json:"help"`
}

func (c *Controller) List() error {
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
			return fmt.Errorf("read a SKILL.md file: %w", err)
		}
		result := &docs.Result{}
		if err := docs.Parse(b, result); err != nil {
			return fmt.Errorf("parse a SKILL.md frontmatter: %w", err)
		}
		results = append(results, result)
		return nil
	}); err != nil {
		return fmt.Errorf("walk the skills directory: %w", err)
	}
	encoder := json.NewEncoder(c.stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(&Results{
		Results: results,
		Help:    "Run `ghtkn docs show {name}` to see the details of each document.",
	}); err != nil {
		return fmt.Errorf("encode documents as JSON: %w", err)
	}
	return nil
}
