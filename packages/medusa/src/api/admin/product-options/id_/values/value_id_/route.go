// Package value_id_ mirrors api/admin/product-options/[id]/values/[value_id]/route.ts.
package value_id_

import (
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	productoptions "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/product-options"
)

func retrieveValue(req *medusahttp.MedusaRequest, fields []string) (any, error) {
	if fields == nil {
		fields = []string{"id"}
	}
	f := dal.NewRecord()
	f.Set("id", req.Params["value_id"])
	f.Set("option_id", req.Params["id"])
	v, err := productoptions.GraphOne(req, "product_option_value", f, fields)
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf(`Product option value with id "%s" was not found for option with id "%s"`, req.Params["value_id"], req.Params["id"]))
	}
	return v, nil
}

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	v, err := retrieveValue(req, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	return productoptions.Respond(res, "product_option_value", v)
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	if _, err := retrieveValue(req, nil); err != nil {
		return err
	}
	body, _ := req.ValidatedBody.(*dal.Record)
	if _, err := workflows.UpdateProductOptionValuesWorkflow.Run(req.Ctx(), req.Scope, steps.UpdateProductOptionValuesStepInput{ID: req.Params["value_id"], Update: body}); err != nil {
		return err
	}
	f := dal.NewRecord()
	f.Set("id", req.Params["value_id"])
	v, err := productoptions.GraphOne(req, "product_option_value", f, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	return productoptions.Respond(res, "product_option_value", v)
}

// DELETE mirrors DELETE.
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	if _, err := retrieveValue(req, nil); err != nil {
		return err
	}
	id := req.Params["value_id"]
	if _, err := workflows.DeleteProductOptionValuesWorkflow.Run(req.Ctx(), req.Scope, []any{id}); err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("id", id)
	out.Set("object", "product_option_value")
	out.Set("deleted", true)
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET, POST: POST, DELETE: DELETE}
