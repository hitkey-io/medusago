// Package shippingmethods mirrors api/store/carts/[id]/shipping-methods.
package shippingmethods

import (
	cartworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/store/carts"
)

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	var payload []any
	var additional *dal.Record
	switch t := req.ValidatedBody.(type) {
	case []any:
		payload = t
	case *dal.Record:
		payload = []any{t}
		additional, _ = carts.Get(t, "additional_data").(*dal.Record)
	}
	var options []*dal.Record
	for _, p := range payload {
		r, _ := p.(*dal.Record)
		o := dal.NewRecord()
		o.Set("id", carts.Get(r, "option_id"))
		if v, ok := r.Get("data"); ok {
			o.Set("data", v)
		}
		options = append(options, o)
	}
	if _, err := cartworkflows.AddShippingMethodToCartWorkflow.Run(req.Ctx(), req.Scope, cartworkflows.AddShippingMethodInput{CartID: req.Params["id"], Options: options, AdditionalData: additional}); err != nil {
		return err
	}
	return carts.RespondCart(req, res, req.Params["id"])
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
