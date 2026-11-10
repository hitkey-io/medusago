package draftorders

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
)

// GET mirrors GET /admin/draft-orders.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	filters := dal.NewRecord()
	if req.FilterableFields != nil {
		for _, k := range req.FilterableFields.Keys() {
			v, _ := req.FilterableFields.Get(k)
			filters.Set(k, v)
		}
	}
	filters.Set("is_draft_order", true)
	r, err := workflows.GetOrdersList(req.Ctx(), httputils.QueryOf(req.Scope), workflows.GetOrdersListInput{
		Fields:    req.QueryConfig.Fields,
		Variables: httputils.ListVariables(filters, req.QueryConfig.Pagination),
	})
	if err != nil {
		return err
	}
	if r.Metadata == nil {
		r.Metadata = dal.NewRecord()
	}
	return res.JSON(httputils.ListResponse("draft_orders", r))
}

func val(r *dal.Record, k string) any {
	v, _ := r.Get(k)
	return v
}

// RefetchOrder mirrors refetchOrder.
func RefetchOrder(req *medusahttp.MedusaRequest, id any) (any, error) {
	f := dal.NewRecord()
	f.Set("id", id)
	return httputils.RemoteQueryFirst(req.Ctx(), req.Scope, "order", f, req.QueryConfig.Fields)
}

// POST mirrors POST /admin/draft-orders.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body := orders.BodyCopy(req)
	input := dal.NewRecord()
	for _, k := range body.Keys() {
		v, _ := body.Get(k)
		input.Set(k, v)
	}
	nn, _ := body.Get("no_notification_order")
	b, _ := nn.(bool)
	input.Set("no_notification", b)
	input.Set("status", "draft")
	input.Set("is_draft_order", true)
	if v, _ := input.Get("currency_code"); v == nil || v == "" {
		f := dal.NewRecord()
		f.Set("id", val(body, "region_id"))
		region, err := httputils.RemoteQueryFirst(req.Ctx(), req.Scope, "region", f, []string{"currency_code"})
		if err != nil {
			return err
		}
		var code any
		if r, ok := region.(*dal.Record); ok {
			code, _ = r.Get("currency_code")
		}
		input.Set("currency_code", code)
	}
	if v, _ := input.Get("email"); (v == nil || v == "") && val(body, "customer_id") != nil {
		f := dal.NewRecord()
		f.Set("id", val(body, "customer_id"))
		customer, err := httputils.RemoteQueryFirst(req.Ctx(), req.Scope, "customer", f, []string{"email"})
		if err != nil {
			return err
		}
		var email any
		if r, ok := customer.(*dal.Record); ok {
			email, _ = r.Get("email")
		}
		input.Set("email", email)
	}
	for _, k := range []string{"billing_address", "shipping_address"} {
		if s, ok := val(body, k).(string); ok {
			input.Set(k+"_id", s)
			input.Delete(k)
		}
	}
	order, err := workflows.CreateOrderWorkflow.Run(req.Ctx(), req.Scope, input)
	if err != nil {
		return err
	}
	id, _ := order.Get("id")
	draft, err := RefetchOrder(req, id)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("draft_order", draft)
	return res.Status(200).JSON(out)
}

// Route is route.ts.
var Route = medusahttp.Route{GET: GET, POST: POST}

// RespondDraftPreview writes { draft_order_preview }.
func RespondDraftPreview(res *medusahttp.MedusaResponse, preview *dal.Record) error {
	out := dal.NewRecord()
	out.Set("draft_order_preview", preview)
	return res.JSON(out)
}

// BodyRecords returns the records of a validated body array field.
func BodyRecords(body *dal.Record, key string) []*dal.Record {
	v, _ := body.Get(key)
	list, _ := v.([]any)
	out := make([]*dal.Record, 0, len(list))
	for _, x := range list {
		if r, ok := x.(*dal.Record); ok {
			out = append(out, r)
		}
	}
	return out
}

// BodyStrings returns the strings of a validated body array field.
func BodyStrings(body *dal.Record, key string) []string {
	v, _ := body.Get(key)
	list, _ := v.([]any)
	out := make([]string, 0, len(list))
	for _, x := range list {
		if s, ok := x.(string); ok {
			out = append(out, s)
		}
	}
	return out
}
