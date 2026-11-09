// Package batch mirrors api/admin/products/[id]/variants/inventory-items/batch/route.ts.
package batch

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/common/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/products"
)

func list(rs []*dal.Record) []any {
	out := make([]any, len(rs))
	for i, r := range rs {
		out[i] = r
	}
	return out
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	var create, update, del any
	if body != nil {
		create, _ = body.Get("create")
		update, _ = body.Get("update")
		del, _ = body.Get("delete")
	}
	result, err := workflows.BatchLinksWorkflow.Run(req.Ctx(), req.Scope, workflows.BatchLinksInput{
		Create: products.BuildBatchVariantInventoryData(create),
		Update: products.BuildBatchVariantInventoryData(update),
		Delete: products.BuildBatchVariantInventoryData(del),
	})
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("created", list(result.Created))
	out.Set("updated", list(result.Updated))
	out.Set("deleted", list(result.Deleted))
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{POST: POST}
