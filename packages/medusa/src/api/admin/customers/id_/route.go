// Package id_ mirrors api/admin/customers/[id]/route.ts.
package id_

import (
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/customer/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/customers"
)

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	existing, err := customers.RefetchCustomer(req, id, []string{"id"})
	if err != nil {
		return err
	}
	if existing == nil {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf(`Customer with id "%s" not found`, id))
	}
	body, _ := req.ValidatedBody.(*dal.Record)
	rest, additional := customers.SplitAdditionalData(body)
	selector := dal.NewRecord()
	selector.Set("id", id)
	if _, err := workflows.UpdateCustomersWorkflow.Run(req.Ctx(), req.Scope, workflows.UpdateCustomersInput{Selector: selector, Update: rest, AdditionalData: additional}); err != nil {
		return err
	}
	return customers.Respond(req, res, id)
}

// DELETE mirrors DELETE.
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	if _, err := workflows.RemoveCustomerAccountWorkflow.Run(req.Ctx(), req.Scope, workflows.RemoveCustomerAccountInput{CustomerID: id}); err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("id", id)
	out.Set("object", "customer")
	out.Set("deleted", true)
	return res.Status(200).JSON(out)
}

var Route = medusahttp.Route{GET: httputils.RemoteQueryRetrieveRoute("customer", "customer", "Customer with id: %s not found"), POST: POST, DELETE: DELETE}
