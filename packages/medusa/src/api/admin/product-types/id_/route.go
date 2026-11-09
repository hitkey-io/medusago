// Package id_ mirrors api/admin/product-types/[id]/route.ts.
package id_

import (
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	producttypes "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/product-types"
)

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	entity, err := producttypes.Refetch(req.Ctx(), req.Scope, req.Params["id"], req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	return producttypes.Respond(res, "product_type", entity)
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	existing, err := producttypes.Refetch(req.Ctx(), req.Scope, req.Params["id"], []string{"id"})
	if err != nil {
		return err
	}
	if existing == nil {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf(`Product type with id "%s" not found`, req.Params["id"]))
	}
	body, _ := req.ValidatedBody.(*dal.Record)
	selector := dal.NewRecord()
	selector.Set("id", req.Params["id"])
	result, err := workflows.UpdateProductTypesWorkflow.Run(req.Ctx(), req.Scope, workflows.UpdateProductTypesInput{Selector: selector, Update: body})
	if err != nil {
		return err
	}
	var id0 any = req.Params["id"]
	if len(result) > 0 {
		id0, _ = result[0].Get("id")
	}
	entity, err := producttypes.Refetch(req.Ctx(), req.Scope, id0, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	return producttypes.Respond(res, "product_type", entity)
}

// DELETE mirrors DELETE.
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	if _, err := workflows.DeleteProductTypesWorkflow.Run(req.Ctx(), req.Scope, workflows.DeleteProductTypesInput{IDs: []any{id}}); err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("id", id)
	out.Set("object", "product_type")
	out.Set("deleted", true)
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET, POST: POST, DELETE: DELETE}
