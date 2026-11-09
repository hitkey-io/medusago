// Package products mirrors api/admin/product-categories/[id]/products/route.ts.
package products

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	productcategories "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/product-categories"
)

// LinkInput builds the link input from { add, remove }.
func LinkInput(id string, body *dal.Record) steps.LinkWorkflowInput {
	in := steps.LinkWorkflowInput{ID: id}
	if body != nil {
		if v, ok := body.Get("add"); ok {
			in.Add, _ = v.([]any)
		}
		if v, ok := body.Get("remove"); ok {
			in.Remove, _ = v.([]any)
		}
	}
	return in
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	if _, err := workflows.BatchLinkProductsToCategoryWorkflow.Run(req.Ctx(), req.Scope, LinkInput(req.Params["id"], body)); err != nil {
		return err
	}
	c, err := httputils.RefetchEntity(req.Ctx(), req.Scope, "product_category", req.Params["id"], req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	return productcategories.Respond(res, c)
}

// Route is the route file.
var Route = medusahttp.Route{POST: POST}
