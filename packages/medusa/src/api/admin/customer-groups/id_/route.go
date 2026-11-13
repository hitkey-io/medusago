// Package id_ mirrors api/admin/customer-groups/[id]/route.ts.
package id_

import (
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/customer-group/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/customer-group/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	customergroups "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/customer-groups"
)

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	existing, err := customergroups.RefetchCustomerGroup(req, id, []string{"id"})
	if err != nil {
		return err
	}
	if existing == nil {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf(`Customer group with id "%s" not found`, id))
	}
	body, _ := req.ValidatedBody.(*dal.Record)
	selector := dal.NewRecord()
	selector.Set("id", id)
	if _, err := workflows.UpdateCustomerGroupsWorkflow.Run(req.Ctx(), req.Scope, steps.UpdateCustomerGroupsStepInput{Selector: selector, Update: body}); err != nil {
		return err
	}
	return customergroups.Respond(req, res, id)
}

// DELETE mirrors DELETE.
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	if _, err := workflows.DeleteCustomerGroupsWorkflow.Run(req.Ctx(), req.Scope, workflows.DeleteCustomerGroupsInput{IDs: []any{id}}); err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("id", id)
	out.Set("object", "customer_group")
	out.Set("deleted", true)
	return res.Status(200).JSON(out)
}

var Route = medusahttp.Route{GET: httputils.RemoteQueryRetrieveRoute("customer_group", "customer_group", "Customer group with id: %s not found"), POST: POST, DELETE: DELETE}
