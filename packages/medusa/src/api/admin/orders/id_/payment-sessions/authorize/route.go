// Package authorize mirrors api/admin/orders/[id]/payment-sessions/authorize.
package authorize

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
)

// POST mirrors POST /admin/orders/:id/payment-sessions/authorize.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	sid, _ := orders.BodyCopy(req).Get("payment_session_id")
	s, _ := sid.(string)
	payment, err := workflows.AuthorizePaymentSessionForOrderWorkflow.Run(req.Ctx(), req.Scope, s)
	if err != nil {
		return err
	}
	order, err := workflows.GetOrderDetail(req.Ctx(), req.Scope, workflows.GetOrderDetailInput{OrderID: req.Params["id"], Fields: req.QueryConfig.Fields})
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("order", order)
	out.Set("is_authorized", payment != nil)
	return res.Status(200).JSON(out)
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
