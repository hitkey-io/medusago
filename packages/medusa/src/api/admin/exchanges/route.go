package exchanges

import (
	"context"
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/returns"
)

// GET mirrors GET /admin/exchanges.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	r, err := httputils.RemoteQueryList(req.Ctx(), req.Scope, "order_exchanges", httputils.ListVariables(req.FilterableFields, req.QueryConfig.Pagination), req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	return res.JSON(httputils.ListResponse("exchanges", r))
}

type orderRetriever interface {
	RetrieveOrder(ctx context.Context, id string, config utilsmodulessdk.FindConfig) (*dal.Record, error)
}

func filters(req *medusahttp.MedusaRequest) *dal.Record {
	f := dal.NewRecord()
	if req.FilterableFields != nil {
		for _, k := range req.FilterableFields.Keys() {
			x, _ := req.FilterableFields.Get(k)
			f.Set(k, x)
		}
	}
	return f
}

// QueryExchange mirrors remoteQuery({ entryPoint: "order_exchange", variables: { id,
// filters }, fields: req.queryConfig.fields })[0].
func QueryExchange(req *medusahttp.MedusaRequest, id any) (any, error) {
	v := dal.NewRecord()
	v.Set("id", id)
	v.Set("filters", filters(req))
	r, err := httputils.RemoteQueryList(req.Ctx(), req.Scope, "order_exchange", v, req.QueryConfig.Fields)
	if err != nil {
		return nil, err
	}
	if len(r.Data) == 0 {
		return nil, nil
	}
	return r.Data[0], nil
}

// QueryReturnDetails mirrors remoteQuery({ entryPoint: "return", variables:
// { id }, fields: defaultAdminDetailsReturnFields })[0].
func QueryReturnDetails(req *medusahttp.MedusaRequest, id any) (any, error) {
	v := dal.NewRecord()
	v.Set("id", id)
	r, err := httputils.RemoteQueryList(req.Ctx(), req.Scope, "return", v, returns.DefaultAdminDetailsReturnFields)
	if err != nil {
		return nil, err
	}
	if len(r.Data) == 0 {
		return nil, nil
	}
	return r.Data[0], nil
}

// RespondPreview writes { order_preview, exchange }.
func RespondPreview(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, preview *dal.Record) error {
	exchange, err := QueryExchange(req, req.Params["id"])
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("order_preview", preview)
	out.Set("exchange", exchange)
	return res.JSON(out)
}

// RespondReturnPreview writes { order_preview, return } for the inbound routes.
func RespondReturnPreview(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, preview *dal.Record) error {
	change, _ := preview.Get("order_change")
	var returnID any
	if c, ok := change.(*dal.Record); ok {
		returnID, _ = c.Get("return_id")
	}
	ret, err := QueryReturnDetails(req, returnID)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("order_preview", preview)
	out.Set("return", ret)
	return res.JSON(out)
}

// ExchangeReturnID mirrors remoteQuery({ entryPoint: "order_exchange", variables:
// { id }, fields: ["id", "return_id"] }, { throwIfKeyNotFound: true }).
func ExchangeReturnID(req *medusahttp.MedusaRequest) (any, error) {
	v := dal.NewRecord()
	v.Set("id", req.Params["id"])
	r, err := httputils.RemoteQueryListOpts(req.Ctx(), req.Scope, "order_exchange", v, []string{"id", "return_id"}, true)
	if err != nil {
		return nil, err
	}
	if len(r.Data) == 0 {
		return nil, nil
	}
	c, _ := r.Data[0].(*dal.Record)
	if c == nil {
		return nil, nil
	}
	id, _ := c.Get("return_id")
	return id, nil
}

// POST mirrors POST /admin/exchanges.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	input := orders.BodyCopy(req)
	input.Set("created_by", orders.ActorID(req))
	change, err := workflows.BeginExchangeOrderWorkflow.Run(req.Ctx(), req.Scope, input)
	if err != nil {
		return err
	}
	exchangeID, _ := change.Get("exchange_id")
	orderID, _ := change.Get("order_id")
	order, err := req.Scope.Resolve(utilsmodulessdk.ORDER).(orderRetriever).RetrieveOrder(req.Ctx(), fmt.Sprint(orderID), utilsmodulessdk.FindConfig{})
	if err != nil {
		return err
	}
	exchange, err := QueryExchange(req, exchangeID)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("order", order)
	out.Set("exchange", exchange)
	return res.JSON(out)
}

// Route is route.ts.
var Route = medusahttp.Route{GET: GET, POST: POST}
