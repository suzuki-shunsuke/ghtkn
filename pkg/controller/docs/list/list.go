package list

import (
	"encoding/json"
	"io/fs"

	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/docs"
	"github.com/suzuki-shunsuke/ghtkn/skills"
)

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
			return err
		}
		result := &docs.Result{}
		if err := docs.Parse(b, result); err != nil {
			return err
		}
		results = append(results, result)
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
