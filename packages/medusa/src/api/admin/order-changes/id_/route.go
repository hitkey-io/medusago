// Package id_ mirrors api/admin/order-changes/[id].
package id_

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
)

// POST mirrors POST /admin/order-changes/:id.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	b := orders.BodyCopy(req)
	in := dal.NewRecord()
	in.Set("id", req.Params["id"])
	for _, k := range []string{"carry_over_promotions", "internal_note"} {
		if v, ok := b.Get(k); ok {
			in.Set(k, v)
		}
	}
	if _, err := workflows.UpdateOrderChangeWorkflow.Run(req.Ctx(), req.Scope, in); err != nil {
		return err
	}
	f := dal.NewRecord()
	if req.FilterableFields != nil {
		for _, k := range req.FilterableFields.Keys() {
			v, _ := req.FilterableFields.Get(k)
			f.Set(k, v)
		}
	}
	f.Set("id", req.Params["id"])
	r, err := httputils.QueryOf(req.Scope).Graph(req.Ctx(), query.GraphInput{Entity: "order_change", Filters: f, Fields: req.QueryConfig.Fields}, joiner.Options{})
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	if len(r.Data) > 0 {
		out.Set("order_change", r.Data[0])
	}
	return res.Status(200).JSON(out)
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
