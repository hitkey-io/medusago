// Package line_id_ mirrors api/store/carts/[id]/line-items/[line_id].
package line_id_

import (
	cartworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/store/carts"
)

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	additional, _ := carts.Get(body, "additional_data").(*dal.Record)
	if _, err := cartworkflows.UpdateLineItemInCartWorkflow.Run(req.Ctx(), req.Scope, cartworkflows.UpdateLineItemInCartInput{CartID: req.Params["id"], ItemID: req.Params["line_id"], Update: body, AdditionalData: additional}, composer.RunOptions{Notify: true}); err != nil {
		return err
	}
	return carts.RespondCart(req, res, req.Params["id"])
}

// DELETE mirrors DELETE.
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["line_id"]
	if _, err := cartworkflows.DeleteLineItemsWorkflow.Run(req.Ctx(), req.Scope, cartworkflows.DeleteLineItemsInput{CartID: req.Params["id"], IDs: []any{id}}, composer.RunOptions{Notify: true}); err != nil {
		return err
	}
	cart, err := carts.RefetchCart(req, req.Params["id"])
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("id", id)
	out.Set("object", "line-item")
	out.Set("deleted", true)
	out.Set("parent", cart)
	return res.Status(200).JSON(out)
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST, DELETE: DELETE}
