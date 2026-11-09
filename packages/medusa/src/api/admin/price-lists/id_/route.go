// Package id_ mirrors api/admin/price-lists/[id]/route.ts.
package id_

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/price-list/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	pricelists "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/price-lists"
)

// FetchPriceList mirrors fetchPriceList.
func FetchPriceList(req *medusahttp.MedusaRequest, id string, fields []string) (any, error) {
	return pricelists.FetchPriceList(req.Ctx(), req.Scope, id, fields)
}

func respond(res *medusahttp.MedusaResponse, pl any) error {
	out := dal.NewRecord()
	out.Set("price_list", pl)
	return res.Status(200).JSON(out)
}

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	pl, err := FetchPriceList(req, req.Params["id"], req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	return respond(res, pl)
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	update := dal.NewRecord()
	if body, ok := req.ValidatedBody.(*dal.Record); ok {
		for _, k := range body.Keys() {
			v, _ := body.Get(k)
			update.Set(k, v)
		}
	}
	update.Set("id", id)
	if _, err := workflows.UpdatePriceListsWorkflow.Run(req.Ctx(), req.Scope, []*dal.Record{update}); err != nil {
		return err
	}
	pl, err := FetchPriceList(req, id, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	return respond(res, pl)
}

// DELETE mirrors DELETE.
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	if _, err := workflows.DeletePriceListsWorkflow.Run(req.Ctx(), req.Scope, []any{id}); err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("id", id)
	out.Set("object", "price_list")
	out.Set("deleted", true)
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET, POST: POST, DELETE: DELETE}
