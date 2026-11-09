package collections

import (
	"context"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// Refetch mirrors the refetch helper: remoteQuery with filters <built-in function id>
// (undefined when missing).
func Refetch(ctx context.Context, scope medusahttp.MedusaContainer, id any, fields []string) (any, error) {
	filters := dal.NewRecord()
	filters.Set("id", id)
	variables := dal.NewRecord()
	variables.Set("filters", filters)
	r, err := httputils.RemoteQueryList(ctx, scope, "product_collection", variables, fields)
	if err != nil {
		return nil, err
	}
	if len(r.Data) == 0 {
		return nil, nil
	}
	return r.Data[0], nil
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
