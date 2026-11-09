// Package batch mirrors api/admin/price-lists/[id]/prices/batch/route.ts.
package batch

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/price-list/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	pricelists "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/price-lists"
)

func records(v any) []*dal.Record {
	list, _ := v.([]any)
	out := []*dal.Record{}
	for _, x := range list {
		if r, ok := x.(*dal.Record); ok {
			out = append(out, r)
		}
	}
	return out
}

func ids(rs []*dal.Record) []any {
	out := []any{}
	for _, r := range rs {
		v, _ := r.Get("id")
		out = append(out, v)
	}
	return out
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	var create, update, del any
	if body != nil {
		create, _ = body.Get("create")
		update, _ = body.Get("update")
		del, _ = body.Get("delete")
	}
	deleteIDs, _ := del.([]any)
	if deleteIDs == nil {
		deleteIDs = []any{}
	}
	result, err := workflows.BatchPriceListPricesWorkflow.Run(req.Ctx(), req.Scope, workflows.BatchPriceListPricesInput{
		ID: req.Params["id"], Create: records(create), Update: records(update), Delete: deleteIDs,
	})
	if err != nil {
		return err
	}
	created, err := pricelists.ListPrices(req.Ctx(), req.Scope, ids(result.Created), pricelists.AdminPriceListPriceQueryFields)
	if err != nil {
		return err
	}
	updated, err := pricelists.ListPrices(req.Ctx(), req.Scope, ids(result.Updated), pricelists.AdminPriceListPriceQueryFields)
	if err != nil {
		return err
	}
	deleted := dal.NewRecord()
	deleted.Set("ids", deleteIDs)
	deleted.Set("object", "price")
	deleted.Set("deleted", true)
	out := dal.NewRecord()
	out.Set("created", created)
	out.Set("updated", updated)
	out.Set("deleted", deleted)
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{POST: POST}
