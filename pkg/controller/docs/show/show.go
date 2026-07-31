package show

// ghtkn docs show backend
// docs/
//   backend.md

import (
	"fmt"
	"strings"

	docsfs "github.com/suzuki-shunsuke/ghtkn/docs"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/docs"
)

func (c *Controller) Show(docName string) error {
	b, err := docsfs.FS.ReadFile(docName + docs.Ext)
	if err != nil {
		return notFoundError(docName)
	}
	fmt.Fprintln(c.stdout, string(b))
	return nil
}

// notFoundError tells which documents are available, so that a coding agent that
// guessed a wrong name can recover without running `ghtkn docs list` again.
func notFoundError(docName string) error {
	names, err := docs.Names(docsfs.FS)
	if err != nil {
		return fmt.Errorf("the document %s isn't found: %w", docName, err)
	}
	return fmt.Errorf("the document %s isn't found. Available documents: %s", docName, strings.Join(names, ", "))
}
