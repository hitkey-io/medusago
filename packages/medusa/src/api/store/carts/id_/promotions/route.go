// Package promotions mirrors api/store/carts/[id]/promotions.
package promotions

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/steps"
	cartworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/store/carts"
)

func codes(req *medusahttp.MedusaRequest) []string {
	body, _ := req.ValidatedBody.(*dal.Record)
	out := []string{}
	if list, ok := carts.Get(body, "promo_codes").([]any); ok {
		for _, c := range list {
			if s, ok := c.(string); ok {
				out = append(out, s)
			}
		}
	}
	return out
}

func run(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, action string) error {
	c := codes(req)
	if action == "" {
		action = steps.PromotionActionReplace
		if len(c) > 0 {
			action = steps.PromotionActionAdd
		}
	}
	if _, err := cartworkflows.UpdateCartPromotionsWorkflow.Run(req.Ctx(), req.Scope, cartworkflows.UpdateCartPromotionsInput{CartID: req.Params["id"], PromoCodes: c, Action: action, ForceRefreshPaymentCollection: true}, composer.RunOptions{Notify: true}); err != nil {
		return err
	}
	return carts.RespondCart(req, res, req.Params["id"])
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	return run(req, res, "")
}

// DELETE mirrors DELETE.
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	return run(req, res, steps.PromotionActionRemove)
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST, DELETE: DELETE}
