// Package creditlines mirrors api/admin/orders/[id]/credit-lines.
package creditlines

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
)

// POST mirrors POST /admin/orders/:id/credit-lines.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	if _, err := workflows.CreateOrderCreditLinesWorkflow.Run(req.Ctx(), req.Scope, struct {
		ID          any
		CreditLines []*dal.Record
	}{id, []*dal.Record{orders.BodyCopy(req)}}); err != nil {
		return err
	}
	f := dal.NewRecord()
	f.Set("id", id)
	r, err := httputils.QueryOf(req.Scope).Graph(req.Ctx(), query.GraphInput{Entity: "orders", Fields: req.QueryConfig.Fields, Filters: f}, joiner.Options{ThrowIfKeyNotFound: true})
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	if len(r.Data) > 0 {
		out.Set("order", r.Data[0])
	}
	return res.Status(200).JSON(out)
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
