// Package id_ mirrors api/admin/products/[id]/route.ts.
package id_

import (
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/products"
)

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	p, err := httputils.RefetchEntity(req.Ctx(), req.Scope, "product", req.Params["id"], products.RemapKeysForProduct(req.QueryConfig.Fields))
	if err != nil {
		return err
	}
	if p == nil {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, "Product not found")
	}
	out := dal.NewRecord()
	out.Set("product", products.RemapProductResponse(p))
	return res.Status(200).JSON(out)
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	update := dal.NewRecord()
	var additional *dal.Record
	for _, k := range body.Keys() {
		v, _ := body.Get(k)
		if k == "additional_data" {
			additional, _ = v.(*dal.Record)
			continue
		}
		update.Set(k, v)
	}
	existing, err := httputils.RefetchEntity(req.Ctx(), req.Scope, "product", req.Params["id"], []string{"id"})
	if err != nil {
		return err
	}
	if existing == nil {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf(`Product with id "%s" not found`, req.Params["id"]))
	}
	selector := dal.NewRecord()
	selector.Set("id", req.Params["id"])
	result, err := workflows.UpdateProductsWorkflow.Run(req.Ctx(), req.Scope, workflows.UpdateProductsInput{Selector: selector, Update: update, AdditionalData: additional})
	if err != nil {
		return err
	}
	id, _ := result[0].Get("id")
	p, err := httputils.RefetchEntity(req.Ctx(), req.Scope, "product", id, products.RemapKeysForProduct(req.QueryConfig.Fields))
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("product", products.RemapProductResponse(p))
	return res.Status(200).JSON(out)
}

// DELETE mirrors DELETE.
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	if _, err := workflows.DeleteProductsWorkflow.Run(req.Ctx(), req.Scope, workflows.DeleteProductsInput{IDs: []any{id}}); err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("id", id)
	out.Set("object", "product")
	out.Set("deleted", true)
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET, POST: POST, DELETE: DELETE}
