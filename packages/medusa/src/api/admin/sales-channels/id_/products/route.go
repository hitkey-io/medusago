// Package products mirrors api/admin/sales-channels/[id]/products/route.ts.
package products

import (
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/sales-channel/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	saleschannels "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/sales-channels"
)

func strs(v any) []string {
	l, _ := v.([]any)
	if l == nil {
		return nil
	}
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
	if _, err := workflows.LinkProductsToSalesChannelWorkflow.Run(req.Ctx(), req.Scope, workflows.LinkProductsInput{ID: req.Params["id"], Add: strs(add), Remove: strs(remove)}); err != nil {
		return err
	}
	v, err := saleschannels.RefetchSalesChannel(req, req.Params["id"], req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	return saleschannels.Respond(res, v)
}

// Route is the route file.
var Route = medusahttp.Route{POST: POST}
