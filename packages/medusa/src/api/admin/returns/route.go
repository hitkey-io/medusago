package returns

import (
	"context"
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
)

// GET mirrors GET /admin/returns.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	r, err := httputils.RemoteQueryList(req.Ctx(), req.Scope, "returns", httputils.ListVariables(req.FilterableFields, req.QueryConfig.Pagination), req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	return res.JSON(httputils.ListResponse("returns", r))
}

type orderRetriever interface {
	RetrieveOrder(ctx context.Context, id string, config utilsmodulessdk.FindConfig) (*dal.Record, error)
}

// QueryReturn mirrors remoteQuery({ entryPoint: "return", variables: { id,
// filters }, fields }) and returns the first row.
func QueryReturn(req *medusahttp.MedusaRequest, id any) (any, error) {
	v := dal.NewRecord()
	v.Set("id", id)
	filters := dal.NewRecord()
	if req.FilterableFields != nil {
		for _, k := range req.FilterableFields.Keys() {
			x, _ := req.FilterableFields.Get(k)
			filters.Set(k, x)
		}
	}
	v.Set("filters", filters)
	r, err := httputils.RemoteQueryList(req.Ctx(), req.Scope, "return", v, req.QueryConfig.Fields)
	if err != nil {
		return nil, err
	}
	if len(r.Data) == 0 {
		return nil, nil
	}
	return r.Data[0], nil
}

// RespondPreview writes { order_preview, return }.
func RespondPreview(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, preview *dal.Record) error {
	ret, err := QueryReturn(req, req.Params["id"])
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("order_preview", preview)
	out.Set("return", ret)
	return res.JSON(out)
}

// RespondOrderReturn writes { order, return } for an order change result.
func RespondOrderReturn(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, change *dal.Record) error {
	returnID, _ := change.Get("return_id")
	orderID, _ := change.Get("order_id")
	order, err := req.Scope.Resolve(utilsmodulessdk.ORDER).(orderRetriever).RetrieveOrder(req.Ctx(), fmt.Sprint(orderID), utilsmodulessdk.FindConfig{})
	if err != nil {
		return err
	}
	ret, err := QueryReturn(req, returnID)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("order", order)
	out.Set("return", ret)
	return res.JSON(out)
}

// POST mirrors POST /admin/returns.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	input := orders.BodyCopy(req)
	input.Set("created_by", orders.ActorID(req))
	change, err := workflows.BeginReturnOrderWorkflow.Run(req.Ctx(), req.Scope, input)
	if err != nil {
		return err
	}
	return RespondOrderReturn(req, res, change)
}

// Route is route.ts.
var Route = medusahttp.Route{GET: GET, POST: POST}
