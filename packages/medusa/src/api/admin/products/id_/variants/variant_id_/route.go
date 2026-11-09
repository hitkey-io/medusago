// Package variant_id_ mirrors api/admin/products/[id]/variants/[variant_id]/route.ts.
package variant_id_

import (
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/products"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/products/id_/variants"
)

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	filter := dal.NewRecord()
	filter.Set("id", req.Params["variant_id"])
	filter.Set("product_id", req.Params["id"])
	v, err := httputils.RefetchEntity(req.Ctx(), req.Scope, "variant", filter, products.RemapKeysForVariant(req.QueryConfig.Fields))
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	if v != nil {
		out.Set("variant", products.RemapVariantResponse(v))
	}
	return res.Status(200).JSON(out)
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	update, additional := variants.SplitAdditional(body)
	selector := dal.NewRecord()
	selector.Set("id", req.Params["variant_id"])
	selector.Set("product_id", req.Params["id"])
	if _, err := workflows.UpdateProductVariantsWorkflow.Run(req.Ctx(), req.Scope, workflows.UpdateProductVariantsInput{Selector: selector, Update: update, AdditionalData: additional}); err != nil {
		return err
	}
	return variants.RespondProduct(req, res, req.Params["id"])
}

// DELETE mirrors DELETE.
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	productID, variantID := req.Params["id"], req.Params["variant_id"]
	v, err := httputils.RefetchEntity(req.Ctx(), req.Scope, "variant", variantID, []string{"id", "product_id"})
	if err != nil {
		return err
	}
	if r, ok := v.(*dal.Record); ok && r != nil {
		if pid, _ := r.Get("product_id"); fmt.Sprint(pid) != productID {
			return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("Variant with id: %s was not found on product with id: %s", variantID, productID))
		}
	}
	if _, err := workflows.DeleteProductVariantsWorkflow.Run(req.Ctx(), req.Scope, workflows.DeleteProductVariantsInput{IDs: []any{variantID}}); err != nil {
		return err
	}
	p, err := httputils.RefetchEntity(req.Ctx(), req.Scope, "product", productID, products.RemapKeysForProduct(req.QueryConfig.Fields))
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("id", variantID)
	out.Set("object", "variant")
	out.Set("deleted", true)
	if p != nil {
		out.Set("parent", products.RemapProductResponse(p))
	}
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET, POST: POST, DELETE: DELETE}
