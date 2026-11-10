// Package batch mirrors api/admin/products/batch.
package batch

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/products"
)

func recordList(v any) []*dal.Record {
	l, _ := v.([]any)
	out := make([]*dal.Record, 0, len(l))
	for _, x := range l {
		if r, ok := x.(*dal.Record); ok {
			out = append(out, r)
		}
	}
	return out
}

// refetch mirrors one half of refetchBatchProducts.
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
	res, err := httputils.RemoteQueryList(req.Ctx(), req.Scope, "product", vars, products.RemapKeysForProduct(req.QueryConfig.Fields))
	if err != nil {
		return nil, err
	}
	out := make([]any, len(res.Data))
	for i, p := range res.Data {
		out[i] = products.RemapProductResponse(p)
	}
	return out, nil
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	in := workflows.BatchProductsInput{}
	if v, ok := body.Get("create"); ok && v != nil {
		in.Create = recordList(v)
	}
	if v, ok := body.Get("update"); ok && v != nil {
		in.Update = recordList(v)
	}
	if v, ok := body.Get("delete"); ok && v != nil {
		in.Delete, _ = v.([]any)
	}
	result, err := workflows.BatchProductsWorkflow.Run(req.Ctx(), req.Scope, in)
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
	deleted.Set("ids", result.Deleted)
	deleted.Set("object", "product")
	deleted.Set("deleted", true)
	out := dal.NewRecord()
	out.Set("created", created)
	out.Set("updated", updated)
	out.Set("deleted", deleted)
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{POST: POST}
