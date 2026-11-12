// Package complete mirrors api/store/carts/[id]/complete.
package complete

import (
	cartworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/store/carts"
)

// POST mirrors POST /store/carts/:id/complete.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	cartID := req.Params["id"]
	result, err := cartworkflows.CompleteCartWorkflow.Run(req.Ctx(), req.Scope, cartworkflows.CompleteCartWorkflowInput{ID: cartID}, composer.RunOptions{Notify: true})
	if err != nil {
		prepared, perr := httputils.PrepareRetrieveQuery(dal.NewRecord(), httputils.QueryConfig{Defaults: carts.DefaultStoreCartFields}, nil)
		if perr != nil {
			return perr
		}
		f := dal.NewRecord()
		f.Set("id", cartID)
		v := dal.NewRecord()
		v.Set("filters", f)
		r, qerr := httputils.RemoteQueryList(req.Ctx(), req.Scope, "cart", v, prepared.RemoteQueryConfig.Fields)
		if qerr != nil {
			return qerr
		}
		var cart any
		if len(r.Data) > 0 {
			cart = r.Data[0]
		}
		merr, ok := err.(*common.MedusaError)
		if !ok || (merr.Type != common.MedusaErrorTypes.PAYMENT_AUTHORIZATION_ERROR && merr.Type != common.MedusaErrorTypes.PAYMENT_REQUIRES_MORE_ERROR) {
			return err
		}
		e := dal.NewRecord()
		e.Set("message", merr.Message)
		e.Set("name", "Error")
		e.Set("type", merr.Type)
		out := dal.NewRecord()
		out.Set("type", "cart")
		out.Set("cart", cart)
		out.Set("error", e)
		return res.Status(200).JSON(out)
	}
	f := dal.NewRecord()
	id, _ := result.Get("id")
	f.Set("id", id)
	data, err := httputils.QueryOf(req.Scope).Graph(req.Ctx(), query.GraphInput{Entity: "order", Fields: req.QueryConfig.Fields, Filters: f}, joiner.Options{})
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("type", "order")
	if len(data.Data) > 0 {
		out.Set("order", data.Data[0])
	}
	return res.Status(200).JSON(out)
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
