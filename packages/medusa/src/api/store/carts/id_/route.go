// Package id_ mirrors api/store/carts/[id].
package id_

import (
	cartworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/store/carts"
)

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	cart, err := carts.RefetchCart(req, req.Params["id"])
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("cart", cart)
	return res.JSON(out)
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	in := dal.NewRecord()
	if body != nil {
		for _, k := range body.Keys() {
			v, _ := body.Get(k)
			in.Set(k, v)
		}
	}
	in.Set("id", req.Params["id"])
	in.Set("additional_data", carts.Get(body, "additional_data"))
	if _, err := cartworkflows.UpdateCartWorkflow.Run(req.Ctx(), req.Scope, in); err != nil {
		return err
	}
	return carts.RespondCart(req, res, req.Params["id"])
}

// Route is route.ts.
var Route = medusahttp.Route{GET: GET, POST: POST}
