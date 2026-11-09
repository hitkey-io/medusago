// Package products mirrors api/admin/collections/[id]/products/route.ts.
package products

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/collections"
	categoryproducts "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/product-categories/id_/products"
)

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	in := categoryproducts.LinkInput(req.Params["id"], body)
	if in.Add == nil {
		in.Add = []any{}
	}
	if in.Remove == nil {
		in.Remove = []any{}
	}
	if _, err := workflows.BatchLinkProductsToCollectionWorkflow.Run(req.Ctx(), req.Scope, in); err != nil {
		return err
	}
	c, err := collections.Refetch(req.Ctx(), req.Scope, req.Params["id"], req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("collection", c)
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{POST: POST}
