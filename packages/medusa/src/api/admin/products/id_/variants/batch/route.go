// Package batch mirrors api/admin/products/[id]/variants/batch/route.ts.
package batch

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/products"
)

func withProduct(v any, productID string) []*dal.Record {
	list, _ := v.([]any)
	out := make([]*dal.Record, 0, len(list))
	for _, x := range list {
		r, _ := x.(*dal.Record)
		if r == nil {
			continue
		}
		c := dal.NewRecord()
		for _, k := range r.Keys() {
			val, _ := r.Get(k)
			c.Set(k, val)
		}
		c.Set("product_id", productID)
		out = append(out, c)
	}
	return out
}

func refetch(req *medusahttp.MedusaRequest, rows []*dal.Record) ([]any, error) {
	if len(rows) == 0 {
		return []any{}, nil
	}
	ids := make([]any, len(rows))
	for i, r := range rows {
		ids[i], _ = r.Get("id")
	}
	filters := dal.NewRecord()
	filters.Set("id", ids)
	vars := dal.NewRecord()
	vars.Set("filters", filters)
	res, err := httputils.RemoteQueryList(req.Ctx(), req.Scope, "variant", vars, products.RemapKeysForVariant(req.QueryConfig.Fields))
	if err != nil {
		return nil, err
	}
	out := make([]any, len(res.Data))
	for i, v := range res.Data {
		out[i] = products.RemapVariantResponse(v)
	}
	return out, nil
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	productID := req.Params["id"]
	in := workflows.BatchProductVariantsInput{}
	if v, ok := body.Get("create"); ok && v != nil {
		in.Create = withProduct(v, productID)
	}
	if v, ok := body.Get("update"); ok && v != nil {
		in.Update = withProduct(v, productID)
	}
	if v, ok := body.Get("delete"); ok && v != nil {
		in.Delete, _ = v.([]any)
	}
	result, err := workflows.BatchProductVariantsWorkflow.Run(req.Ctx(), req.Scope, in)
	if err != nil {
		return err
	}
	created, err := refetch(req, result.Created)
	if err != nil {
		return err
	}
	updated, err := refetch(req, result.Updated)
	if err != nil {
		return err
	}
	deleted := dal.NewRecord()
	ids := result.Deleted
	if ids == nil {
		ids = []any{}
	}
	deleted.Set("ids", ids)
	deleted.Set("object", "variant")
	deleted.Set("deleted", true)
	out := dal.NewRecord()
	out.Set("created", created)
	out.Set("updated", updated)
	out.Set("deleted", deleted)
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{POST: POST}
