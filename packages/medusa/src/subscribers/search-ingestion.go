package subscribers

import (
	"context"

	coremodulessdk "github.com/hitkey-io/medusago/packages/core/modules-sdk/src"
	searchtypes "github.com/hitkey-io/medusago/packages/core/types/src/search"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

type searchIngester interface {
	Ingest(ctx context.Context, event searchtypes.Event) ([]searchtypes.Task, error)
}

// SearchIngestionHandler mirrors searchIngestionHandler: the event is
// handed to the Search Module's ingest.
func SearchIngestionHandler(ctx context.Context, c composer.Container, name string, data any, metadata *dal.Record) error {
	searchModule, _ := c.Resolve(utilsmodulessdk.SEARCH).(searchIngester)
	if searchModule == nil {
		return nil
	}
	_, err := searchModule.Ingest(ctx, searchtypes.Event{Name: name, Data: data, Metadata: metadata})
	return err
}

// SearchIngestionEvents mirrors the subscriber config: the events the
// registered search index definitions declare (deduplicated).
func SearchIngestionEvents() []string {
	seen := map[string]bool{}
	var out []string
	for _, index := range coremodulessdk.GetSearchIndexes() {
		for _, event := range index.Events {
			if !seen[event] {
				seen[event] = true
				out = append(out, event)
			}
		}
	}
	return out
}
