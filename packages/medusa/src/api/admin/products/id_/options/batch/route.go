// Package batch mirrors api/admin/products/[id]/options/batch/route.ts.
package batch

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/products/id_/variants"
	"github.com/hitkey-io/medusago/packages/modules/product/src/services"
)

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	in := workflows.LinkProductOptionsInput{ProductID: req.Params["id"]}
	if v, ok := body.Get("add"); ok {
		in.Add, _ = v.([]any)
	}
	if v, ok := body.Get("remove"); ok {
		l, _ := v.([]any)
		for _, x := range l {
			if s, ok := x.(string); ok {
				in.Remove = append(in.Remove, s)
			}
		}
	}
	if v, ok := body.Get("update"); ok {
		l, _ := v.([]any)
		for _, x := range l {
			r, _ := x.(*dal.Record)
			if r == nil {
				continue
			}
			u := services.ProductOptionProductValueUpdate{}
			id, _ := r.Get("product_option_id")
			u.ProductOptionID, _ = id.(string)
			add, _ := r.Get("add")
			u.Add, _ = add.([]any)
			rem, _ := r.Get("remove")
			u.Remove, _ = rem.([]any)
			in.Update = append(in.Update, u)
		}
	}
	if _, err := workflows.CreateAndLinkProductOptionsToProductWorkflow.Run(req.Ctx(), req.Scope, in); err != nil {
		return err
	}
	return variants.RespondProduct(req, res, req.Params["id"])
}

// Route is the route file.
var Route = medusahttp.Route{POST: POST}
