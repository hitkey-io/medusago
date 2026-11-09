package configurations

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

func orDefault(meta *dal.Record, key string, def float64) any {
	if v, ok := meta.Get(key); ok && v != nil {
		return v
	}
	return def
}

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	filters := dal.NewRecord()
	if req.FilterableFields != nil {
		for _, k := range req.FilterableFields.Keys() {
			v, _ := req.FilterableFields.Get(k)
			filters.Set(k, v)
		}
	}
	u := dal.NewRecord()
	u.Set("user_id", req.AuthString("actor_id"))
	d := dal.NewRecord()
	d.Set("is_system_default", true)
	filters.Set("$or", []any{u, d})
	r, err := httputils.QueryOf(req.Scope).Graph(req.Ctx(), query.GraphInput{
		Entity: "layout_configuration", Fields: req.QueryConfig.Fields, Filters: filters, Pagination: req.QueryConfig.Pagination,
	}, joiner.Options{})
	if err != nil {
		return err
	}
	meta := r.Metadata
	if meta == nil {
		meta = dal.NewRecord()
	}
	out := dal.NewRecord()
	out.Set("layout_configurations", r.Data)
	out.Set("count", orDefault(meta, "count", 0))
	out.Set("offset", orDefault(meta, "skip", 0))
	out.Set("limit", orDefault(meta, "take", 20))
	return res.JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET}
