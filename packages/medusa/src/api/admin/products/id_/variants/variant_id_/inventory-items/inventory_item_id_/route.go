// Package inventory_item_id_ mirrors
// api/admin/products/[id]/variants/[variant_id]/inventory-items/[inventory_item_id]/route.ts.
package inventory_item_id_

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/common/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/products"
)

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	variantID, itemID := req.Params["variant_id"], req.Params["inventory_item_id"]
	body, _ := req.ValidatedBody.(*dal.Record)
	qty, _ := body.Get("required_quantity")
	data := dal.NewRecord()
	data.Set("required_quantity", qty)
	if _, err := workflows.UpdateLinksWorkflow.Run(req.Ctx(), req.Scope, []*dal.Record{products.VariantInventoryLink(variantID, itemID, data)}); err != nil {
		return err
	}
	variant, err := products.RefetchVariant(req.Ctx(), variantID, req.Scope, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("variant", variant)
	return res.Status(200).JSON(out)
}

// DELETE mirrors DELETE.
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	variantID, itemID := req.Params["variant_id"], req.Params["inventory_item_id"]
	result, err := workflows.DismissLinksWorkflow.Run(req.Ctx(), req.Scope, []*dal.Record{products.VariantInventoryLink(variantID, itemID, nil)})
	if err != nil {
		return err
	}
	parent, err := products.RefetchVariant(req.Ctx(), variantID, req.Scope, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	if len(result) > 0 {
		out.Set("id", result[0])
	}
	out.Set("object", "variant-inventory-item-link")
	out.Set("deleted", true)
	out.Set("parent", parent)
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{POST: POST, DELETE: DELETE}
