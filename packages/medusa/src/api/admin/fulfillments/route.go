package fulfillments

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/fulfillment/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
)

// RespondFulfillment mirrors refetchFulfillment (remoteQuery "fulfillments"
// with filters { id }) and writes { fulfillment }.
func RespondFulfillment(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, id any) error {
	f := dal.NewRecord()
	f.Set("id", id)
	v := dal.NewRecord()
	v.Set("filters", f)
	r, err := httputils.RemoteQueryList(req.Ctx(), req.Scope, "fulfillments", v, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	if len(r.Data) > 0 {
		out.Set("fulfillment", r.Data[0])
	}
	return res.Status(200).JSON(out)
}

// POST mirrors POST /admin/fulfillments.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	in := orders.BodyCopy(req)
	in.Set("created_by", orders.ActorID(req))
	f, err := workflows.CreateFulfillmentWorkflow.Run(req.Ctx(), req.Scope, in)
	if err != nil {
		return err
	}
	id, _ := f.Get("id")
	return RespondFulfillment(req, res, id)
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
