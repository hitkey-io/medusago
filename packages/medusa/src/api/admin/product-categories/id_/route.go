// Package id_ mirrors api/admin/product-categories/[id]/route.ts.
package id_

import (
	"fmt"

	categoryworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/product-category/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	productcategories "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/product-categories"
)

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	c, err := productcategories.RefetchFirst(req, req.Params["id"])
	if err != nil {
		return err
	}
	if c == nil {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("Product category with id: %s was not found", req.Params["id"]))
	}
	out := dal.NewRecord()
	out.Set("product_category", c)
	return res.JSON(out)
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	rest, additional := productcategories.SplitAdditional(body)
	selector := dal.NewRecord()
	selector.Set("id", req.Params["id"])
	if _, err := categoryworkflows.UpdateProductCategoriesWorkflow.Run(req.Ctx(), req.Scope, categoryworkflows.UpdateProductCategoriesInput{Selector: selector, Update: rest, AdditionalData: additional}); err != nil {
		return err
	}
	c, err := productcategories.RefetchFirst(req, req.Params["id"])
	if err != nil {
		return err
	}
	return productcategories.Respond(res, c)
}

// DELETE mirrors DELETE.
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	if _, err := categoryworkflows.DeleteProductCategoriesWorkflow.Run(req.Ctx(), req.Scope, []any{id}); err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("id", id)
	out.Set("object", "product_category")
	out.Set("deleted", true)
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET, POST: POST, DELETE: DELETE}
