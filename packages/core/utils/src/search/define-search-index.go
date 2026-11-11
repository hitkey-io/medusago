package search

import (
	"runtime"

	modulessdk "github.com/hitkey-io/medusago/packages/core/modules-sdk/src"
	searchtypes "github.com/hitkey-io/medusago/packages/core/types/src/search"
)

// DefineSearchIndex mirrors defineSearchIndex: the definition registers
// itself (keyed by name, its source being the caller's file), and the
// Search Module receives every registered definition when it is loaded.
// Fields come from model.Define(...). A conflicting second definition of
// the same name panics, like the import that throws upstream.
func DefineSearchIndex(definition searchtypes.Definition) *searchtypes.Definition {
	_, file, _, _ := runtime.Caller(1)
	def := definition
	if err := modulessdk.SetSearchIndex(&def, file); err != nil {
		panic(err)
	}
	return &def
}
