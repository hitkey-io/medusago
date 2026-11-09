// Package me mirrors api/store/customers/me.
package me

import (
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/customer/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/store/customers"
)

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := customers.ActorID(req)
	c, err := customers.RefetchCustomer(req, id)
	if err != nil {
		return err
	}
	if c == nil {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("Customer with id: %v was not found", id))
	}
	out := dal.NewRecord()
	out.Set("customer", c)
	return res.JSON(out)
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := customers.ActorID(req)
	sel := dal.NewRecord()
	sel.Set("id", id)
	update, _ := req.ValidatedBody.(*dal.Record)
	if _, err := workflows.UpdateCustomersWorkflow.Run(req.Ctx(), req.Scope, workflows.UpdateCustomersInput{Selector: sel, Update: update}); err != nil {
		return err
	}
	return customers.RespondCustomer(req, res, id)
}

// Route is route.ts.
var Route = medusahttp.Route{GET: GET, POST: POST}
