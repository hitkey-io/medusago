// Package customers mirrors api/admin/customer-groups/[id]/customers/route.ts.
package customers

import (
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/customer-group/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/customer-group/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	customergroups "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/customer-groups"
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
	if _, err := workflows.LinkCustomersToCustomerGroupWorkflow.Run(req.Ctx(), req.Scope, steps.LinkInput{ID: req.Params["id"], Add: strs(add), Remove: strs(remove)}); err != nil {
		return err
	}
	return customergroups.Respond(req, res, req.Params["id"])
}

// Route is the route file.
var Route = medusahttp.Route{POST: POST}
