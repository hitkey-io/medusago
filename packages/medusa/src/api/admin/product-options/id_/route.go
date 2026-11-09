// Package id_ mirrors api/admin/product-options/[id]/route.ts.
package id_

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	productoptions "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/product-options"
)

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	f := dal.NewRecord()
	f.Set("id", req.Params["id"])
	o, err := productoptions.GraphOne(req, "product_option", f, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	return productoptions.Respond(res, "product_option", o)
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	selector := dal.NewRecord()
	selector.Set("id", req.Params["id"])
	result, err := workflows.UpdateProductOptionsWorkflow.Run(req.Ctx(), req.Scope, workflows.UpdateProductOptionsInput{Selector: selector, Update: body})
	if err != nil {
		return err
	}
	id, _ := result[0].Get("id")
	f := dal.NewRecord()
	f.Set("id", id)
	o, err := productoptions.GraphOne(req, "product_option", f, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	return productoptions.Respond(res, "product_option", o)
}

// DELETE mirrors DELETE.
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	if _, err := workflows.DeleteProductOptionsWorkflow.Run(req.Ctx(), req.Scope, workflows.DeleteProductOptionsInput{IDs: []any{id}}); err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("id", id)
	out.Set("object", "product_option")
	out.Set("deleted", true)
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET, POST: POST, DELETE: DELETE}
