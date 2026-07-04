package show

// ghtkn docs show ghtkn-backend
// ghtkn-backend/
//   SKILL.md
//   reference.md

import (
	"fmt"
	"io/fs"
	"strings"

	"github.com/suzuki-shunsuke/ghtkn/skills"
)

func (c *Controller) Show(docName string) error {
	contents := []string{}
	if err := fs.WalkDir(skills.FS, docName, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		b, err := skills.FS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read a document file: %w", err)
		}
		contents = append(contents, path+"\n\n"+string(b))
		return nil
	}); err != nil {
		return fmt.Errorf("walk the skills directory: %w", err)
	}
	fmt.Fprintln(c.stdout, strings.Join(contents, "\n=====\n"))
	return nil
}
