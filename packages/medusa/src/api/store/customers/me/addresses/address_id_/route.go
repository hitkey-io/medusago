// Package address_id_ mirrors api/store/customers/me/addresses/[address_id].
package address_id_

import (
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/customer/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/store/customers"
)

func findAddress(req *medusahttp.MedusaRequest, customerID any, addressID string, fields []string) (any, error) {
	f := dal.NewRecord()
	f.Set("id", addressID)
	f.Set("customer_id", customerID)
	v := dal.NewRecord()
	v.Set("filters", f)
	r, err := httputils.RemoteQueryList(req.Ctx(), req.Scope, "customer_address", v, fields)
	if err != nil {
		return nil, err
	}
	if len(r.Data) == 0 || r.Data[0] == nil {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("Address with id: %s was not found", addressID))
	}
	return r.Data[0], nil
}

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	a, err := findAddress(req, customers.ActorID(req), req.Params["address_id"], req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("address", a)
	return res.Status(200).JSON(out)
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := customers.ActorID(req)
	if _, err := findAddress(req, id, req.Params["address_id"], []string{"id"}); err != nil {
		return err
	}
	sel := dal.NewRecord()
	sel.Set("id", req.Params["address_id"])
	sel.Set("customer_id", id)
	update, _ := req.ValidatedBody.(*dal.Record)
	if _, err := workflows.UpdateCustomerAddressesWorkflow.Run(req.Ctx(), req.Scope, workflows.UpdateCustomersInput{Selector: sel, Update: update}); err != nil {
		return err
	}
	return customers.RespondCustomer(req, res, id)
}

// DELETE mirrors DELETE.
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := customers.ActorID(req)
	if _, err := findAddress(req, id, req.Params["address_id"], []string{"id"}); err != nil {
		return err
	}
	if _, err := workflows.DeleteCustomerAddressesWorkflow.Run(req.Ctx(), req.Scope, workflows.IDsInput{IDs: []any{req.Params["address_id"]}}); err != nil {
		return err
	}
	c, err := customers.RefetchCustomer(req, id)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("id", id)
	out.Set("object", "address")
	out.Set("deleted", true)
	out.Set("parent", c)
	return res.Status(200).JSON(out)
}

// Route is route.ts.
var Route = medusahttp.Route{GET: GET, POST: POST, DELETE: DELETE}
