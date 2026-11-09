// Package address_id_ mirrors api/admin/customers/[id]/addresses/[address_id]/route.ts.
package address_id_

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/customer/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/customers"
)

func selector(req *medusahttp.MedusaRequest) *dal.Record {
	f := dal.NewRecord()
	f.Set("id", req.Params["address_id"])
	f.Set("customer_id", req.Params["id"])
	return f
}

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	v, err := httputils.RemoteQueryFirst(req.Ctx(), req.Scope, "customer_address", selector(req), req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	if v != nil {
		out.Set("address", v)
	}
	return res.Status(200).JSON(out)
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	rest, additional := customers.SplitAdditionalData(body)
	if _, err := workflows.UpdateCustomerAddressesWorkflow.Run(req.Ctx(), req.Scope, workflows.UpdateCustomersInput{Selector: selector(req), Update: rest, AdditionalData: additional}); err != nil {
		return err
	}
	return customers.Respond(req, res, req.Params["id"])
}

// DELETE mirrors DELETE.
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["address_id"]
	if _, err := workflows.DeleteCustomerAddressesWorkflow.Run(req.Ctx(), req.Scope, []any{id}); err != nil {
		return err
	}
	customer, err := customers.RefetchCustomer(req, req.Params["id"], req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("id", id)
	out.Set("object", "customer_address")
	out.Set("deleted", true)
	if customer != nil {
		out.Set("parent", customer)
	}
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET, POST: POST, DELETE: DELETE}
