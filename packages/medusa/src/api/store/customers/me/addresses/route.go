// Package addresses mirrors api/store/customers/me/addresses.
package addresses

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/customer/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/store/customers"
)

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	f := dal.NewRecord()
	if req.FilterableFields != nil {
		for _, k := range req.FilterableFields.Keys() {
			v, _ := req.FilterableFields.Get(k)
			f.Set(k, v)
		}
	}
	f.Set("customer_id", customers.ActorID(req))
	r, err := httputils.RemoteQueryList(req.Ctx(), req.Scope, "customer_address", httputils.ListVariables(f, req.QueryConfig.Pagination), req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	return res.JSON(httputils.ListResponse("addresses", r))
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := customers.ActorID(req)
	a := dal.NewRecord()
	if b, ok := req.ValidatedBody.(*dal.Record); ok {
		for _, k := range b.Keys() {
			v, _ := b.Get(k)
			a.Set(k, v)
		}
	}
	a.Set("customer_id", id)
	if _, err := workflows.CreateCustomerAddressesWorkflow.Run(req.Ctx(), req.Scope, workflows.CreateCustomerAddressesInput{Addresses: []*dal.Record{a}}); err != nil {
		return err
	}
	return customers.RespondCustomer(req, res, id)
}

// Route is route.ts.
var Route = medusahttp.Route{GET: GET, POST: POST}
