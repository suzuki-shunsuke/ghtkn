package list

import (
	"encoding/json"
	"fmt"

	docsfs "github.com/suzuki-shunsuke/ghtkn/docs"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/docs"
)

type Results struct {
	Results []*docs.Result `json:"results"`
	Help    string         `json:"help"`
}

func (c *Controller) List() error {
	names, err := docs.Names(docsfs.FS)
	if err != nil {
		return fmt.Errorf("list documents: %w", err)
	}
	results := make([]*docs.Result, 0, len(names))
	for _, name := range names {
		b, err := docsfs.FS.ReadFile(name + docs.Ext)
		if err != nil {
			return fmt.Errorf("read a document file: %w", err)
		}
		result := &docs.Result{
			Name: name,
		}
		if err := docs.Parse(b, result); err != nil {
			return fmt.Errorf("parse the frontmatter of the document %s: %w", name, err)
		}
		results = append(results, result)
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
