package producttags

import (
	"context"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// Refetch mirrors refetchEntity({ entity: "product_tag", idOrFilter: id, fields }).
func Refetch(ctx context.Context, scope medusahttp.MedusaContainer, id any, fields []string) (any, error) {
	return httputils.RefetchEntity(ctx, scope, "product_tag", id, fields)
}

// Respond writes { <key>: entity } (undefined entities are omitted, as
// JSON.stringify does).
func Respond(res *medusahttp.MedusaResponse, key string, entity any) error {
	out := dal.NewRecord()
	if entity != nil {
		out.Set(key, entity)
	}
	return res.Status(200).JSON(out)
}

// SplitAdditionalData returns the body without additional_data, and
// additional_data.
func SplitAdditionalData(body *dal.Record) (*dal.Record, *dal.Record) {
	rest := dal.NewRecord()
	var additional *dal.Record
	if body == nil {
		return rest, nil
	}
	for _, k := range body.Keys() {
		v, _ := body.Get(k)
		if k == "additional_data" {
			additional, _ = v.(*dal.Record)
			continue
		}
		rest.Set(k, v)
	}
	return rest, additional
}
