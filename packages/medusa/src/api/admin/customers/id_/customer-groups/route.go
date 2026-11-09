// Package customergroups mirrors api/admin/customers/[id]/customer-groups/route.ts.
package customergroups

import (
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/customer-group/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/customer-group/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/customers"
)

func strs(v any) []string {
	l, _ := v.([]any)
	out := make([]string, len(l))
	for i, x := range l {
		out[i] = fmt.Sprint(x)
	}
	return out
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	add, _ := body.Get("add")
	remove, _ := body.Get("remove")
	if _, err := workflows.LinkCustomerGroupsToCustomerWorkflow.Run(req.Ctx(), req.Scope, steps.LinkInput{ID: req.Params["id"], Add: strs(add), Remove: strs(remove)}); err != nil {
		return err
	}
	return customers.Respond(req, res, req.Params["id"])
}

// Route is the route file.
var Route = medusahttp.Route{POST: POST}
