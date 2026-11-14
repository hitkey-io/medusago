package brands

import (
	"context"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// region: filter
// ProductsOfBrand retrieves the products linked to a brand with the given
// name, with their brand.
func ProductsOfBrand(ctx context.Context, scope medusahttp.MedusaContainer, name string) ([]any, error) {
	brandFilter := dal.NewRecord()
	brandFilter.Set("name", name)
	filters := dal.NewRecord()
	filters.Set("brand", brandFilter)

	result, err := httputils.QueryOf(scope).Graph(ctx, query.GraphInput{
		Entity:  "product",
		Fields:  []string{"*", "brand.*"},
		Filters: filters,
	}, joiner.Options{})
	if err != nil {
		return nil, err
	}
	return result.Data, nil
}

// endregion: filter
