package modulessdk

import (
	"fmt"
	"sync"

	searchtypes "github.com/hitkey-io/medusago/packages/core/types/src/search"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
)

// The search index registry of MedusaModule (setSearchIndex /
// getSearchIndexes): keyed by index name, in registration order.
var searchIndexes = struct {
	sync.Mutex
	order  []string
	byName map[string]searchIndexEntry
}{byName: map[string]searchIndexEntry{}}

type searchIndexEntry struct {
	definition *searchtypes.Definition
	source     string
}

// SetSearchIndex mirrors MedusaModule.setSearchIndex: registering the same
// definition again is a no-op, and so is a re-registration from the same
// source; two definitions of one name from different sources conflict.
// source is where the definition is declared ("" for an inline one).
func SetSearchIndex(definition *searchtypes.Definition, source string) error {
	searchIndexes.Lock()
	defer searchIndexes.Unlock()
	existing, ok := searchIndexes.byName[definition.Name]
	if ok && existing.definition == definition {
		return nil
	}
	if ok && existing.source != source {
		describe := func(path string) string {
			if path == "" {
				return "the Search Module's options"
			}
			return path
		}
		return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf(
			`Search index "%s" is defined twice: in %s and %s`, definition.Name, describe(existing.source), describe(source)))
	}
	if !ok {
		searchIndexes.order = append(searchIndexes.order, definition.Name)
	}
	searchIndexes.byName[definition.Name] = searchIndexEntry{definition: definition, source: source}
	return nil
}

// GetSearchIndexes mirrors MedusaModule.getSearchIndexes.
func GetSearchIndexes() []*searchtypes.Definition {
	searchIndexes.Lock()
	defer searchIndexes.Unlock()
	out := make([]*searchtypes.Definition, 0, len(searchIndexes.order))
	for _, name := range searchIndexes.order {
		out = append(out, searchIndexes.byName[name].definition)
	}
	return out
}

// ClearSearchIndexes empties the registry (the search part of
// MedusaModule.clearInstances).
func ClearSearchIndexes() {
	searchIndexes.Lock()
	defer searchIndexes.Unlock()
	searchIndexes.order = nil
	searchIndexes.byName = map[string]searchIndexEntry{}
}
