package search

import (
	"context"
	"fmt"
	"strings"
	"sync"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// fallbackSearchInput mirrors FallbackSearchInput.
type fallbackSearchInput struct {
	q      *string
	entity []string
	skip   float64
	take   float64
}

// searchWithGraphFallback mirrors searchWithGraphFallback: the free-text
// `q` fanned out across the admin search entity registry via query.graph.
func searchWithGraphFallback(ctx context.Context, scope medusahttp.MedusaContainer, in fallbackSearchInput) ([]any, error) {
	names := in.entity
	if len(names) == 0 {
		names = DefaultAdminSearchEntityNames
	}
	configs := make([]*AdminSearchEntityConfig, 0, len(names))
	for _, name := range names {
		config := AdminSearchEntityMap[name]
		if config == nil {
			return nil, common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf(`No search entity registered for "%s". Registered entities: %s`, name, strings.Join(DefaultAdminSearchEntityNames, ", ")))
		}
		configs = append(configs, config)
	}

	q := httputils.QueryOf(scope)
	results := make([]any, len(configs))
	errs := make([]error, len(configs))
	var wg sync.WaitGroup
	for i, config := range configs {
		wg.Add(1)
		go func(i int, config *AdminSearchEntityConfig) {
			defer wg.Done()
			filters := dal.NewRecord()
			if config.Filters != nil {
				filters = config.Filters()
			}
			if in.q != nil && *in.q != "" {
				value := *in.q
				if config.TransformQuery != nil {
					value = config.TransformQuery(value)
				}
				filters.Set("q", value)
			}
			entity := config.GraphEntity
			if entity == "" {
				entity = config.Name
			}
			pagination := dal.NewRecord()
			pagination.Set("skip", in.skip)
			pagination.Set("take", in.take)
			res, err := q.Graph(ctx, query.GraphInput{Entity: entity, Fields: config.Fields, Filters: filters, Pagination: pagination}, joiner.Options{})
			if err != nil {
				errs[i] = err
				return
			}
			group := dal.NewRecord()
			group.Set("entity", config.Name)
			group.Set("data", res.Data)
			group.Set("count", orDefault(res.Metadata, "count", float64(0)))
			group.Set("offset", orDefault(res.Metadata, "skip", in.skip))
			group.Set("limit", orDefault(res.Metadata, "take", in.take))
			results[i] = group
		}(i, config)
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}
	return results, nil
}

// orDefault mirrors `metadata?.key ?? fallback`.
func orDefault(metadata *dal.Record, key string, fallback any) any {
	if metadata == nil {
		return fallback
	}
	if v, ok := metadata.Get(key); ok && v != nil {
		return v
	}
	return fallback
}
