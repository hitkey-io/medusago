package orders

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	filters := dal.NewRecord()
	if req.FilterableFields != nil {
		for _, k := range req.FilterableFields.Keys() {
			v, _ := req.FilterableFields.Get(k)
			filters.Set(k, v)
		}
	}
	filters.Set("is_draft_order", false)
	r, err := workflows.GetOrdersList(req.Ctx(), req.Scope, workflows.GetOrdersListInput{
		Fields:    req.QueryConfig.Fields,
		Variables: httputils.ListVariables(filters, req.QueryConfig.Pagination),
	})
	if err != nil {
		return err
	}
	if r.Metadata == nil {
		r.Metadata = dal.NewRecord()
	}
	return res.JSON(httputils.ListResponse("orders", r))
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET}

// ActorID returns req.auth_context.actor_id.
func ActorID(req *medusahttp.MedusaRequest) any {
	if req.AuthContext == nil {
		return nil
	}
	v, _ := req.AuthContext.Get("actor_id")
	return v
}

// RespondOrder refetches the order with remoteQuery({ entryPoint: "order",
// variables: { id }, fields }) and writes { order }.
func RespondOrder(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, id string) error {
	v := dal.NewRecord()
	v.Set("id", id)
	r, err := httputils.RemoteQueryList(req.Ctx(), req.Scope, "order", v, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	if len(r.Data) > 0 {
		out.Set("order", r.Data[0])
	}
	return res.Status(200).JSON(out)
}

// BodyCopy returns a copy of the validated body.
func BodyCopy(req *medusahttp.MedusaRequest) *dal.Record {
	out := dal.NewRecord()
	if b, ok := req.ValidatedBody.(*dal.Record); ok && b != nil {
		for _, k := range b.Keys() {
			v, _ := b.Get(k)
			out.Set(k, v)
		}
	}
	return out
}
