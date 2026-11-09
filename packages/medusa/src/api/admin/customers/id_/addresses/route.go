// Package addresses mirrors api/admin/customers/[id]/addresses/route.ts.
package addresses

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/customer/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/customers"
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
	filters.Set("customer_id", req.Params["id"])
	r, err := httputils.RemoteQueryList(req.Ctx(), req.Scope, "customer_address", httputils.ListVariables(filters, req.QueryConfig.Pagination), req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	return res.JSON(httputils.ListResponse("addresses", r))
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	rest, additional := customers.SplitAdditionalData(body)
	rest.Set("customer_id", req.Params["id"])
	if _, err := workflows.CreateCustomerAddressesWorkflow.Run(req.Ctx(), req.Scope, workflows.CreateCustomerAddressesInput{Addresses: []*dal.Record{rest}, AdditionalData: additional}); err != nil {
		return err
	}
	return customers.Respond(req, res, req.Params["id"])
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET, POST: POST}
