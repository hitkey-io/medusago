// Package hello checks that the examples module builds against the repository.
package hello

import "github.com/hitkey-io/medusago/packages/core/utils/src/dml"

// Note is a throw-away model used to prove the module wiring.
var Note = dml.Define("note", dml.Schema{
	{Name: "id", Property: dml.Id(dml.IdOptions{Prefix: "note"}).PrimaryKey()},
	{Name: "text", Property: dml.Text()},
})
