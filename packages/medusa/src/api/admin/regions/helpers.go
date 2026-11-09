package regions

import (
	"context"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// RefetchRegion mirrors refetchRegion (nil when missing).
func RefetchRegion(ctx context.Context, scope medusahttp.MedusaContainer, id any, fields []string) (any, error) {
	filters := dal.NewRecord()
	filters.Set("id", id)
	variables := dal.NewRecord()
	variables.Set("filters", filters)
	r, err := httputils.RemoteQueryList(ctx, scope, "region", variables, fields)
	if err != nil {
		return nil, err
	}
	if len(r.Data) == 0 {
		return nil, nil
	}
	return r.Data[0], nil
}

// Respond writes { region } (undefined is omitted).
func Respond(res *medusahttp.MedusaResponse, entity any) error {
	out := dal.NewRecord()
	if entity != nil {
		out.Set("region", entity)
	}
	return res.Status(200).JSON(out)
}
