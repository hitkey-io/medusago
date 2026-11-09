// Package batch mirrors api/admin/shipping-options/[id]/rules/batch/route.ts.
package batch

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/fulfillment/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

func records(v any) []*dal.Record {
	list, ok := v.([]any)
	if !ok {
		return nil
	}
	out := []*dal.Record{}
	for _, x := range list {
		if r, ok := x.(*dal.Record); ok {
			out = append(out, r)
		}
	}
	return out
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	var in workflows.BatchShippingOptionRulesInput
	if body, ok := req.ValidatedBody.(*dal.Record); ok {
		if v, ok := body.Get("create"); ok {
			for _, c := range records(v) {
				n := dal.NewRecord()
				for _, k := range c.Keys() {
					x, _ := c.Get(k)
					n.Set(k, x)
				}
				n.Set("shipping_option_id", id)
				in.Create = append(in.Create, n)
			}
		}
		if v, ok := body.Get("update"); ok {
			in.Update = records(v)
		}
		if v, ok := body.Get("delete"); ok {
			in.Delete, _ = v.([]any)
		}
	}
	result, err := workflows.BatchShippingOptionRulesWorkflow.Run(req.Ctx(), req.Scope, in)
	if err != nil {
		return err
	}
	fetch := func(rows []*dal.Record) ([]any, error) {
		if len(rows) == 0 {
			return []any{}, nil
		}
		ids := make([]any, len(rows))
		for i, r := range rows {
			ids[i], _ = r.Get("id")
		}
		f := dal.NewRecord()
		f.Set("id", ids)
		r, err := httputils.RefetchEntities(req.Ctx(), req.Scope, httputils.RefetchEntitiesInput{Entity: "shipping_option_rule", IDOrFilter: f, Fields: req.QueryConfig.Fields})
		if err != nil {
			return nil, err
		}
		if r.Data == nil {
			return []any{}, nil
		}
		return r.Data, nil
	}
	created, err := fetch(result.Created)
	if err != nil {
		return err
	}
	updated, err := fetch(result.Updated)
	if err != nil {
		return err
	}
	deleted := dal.NewRecord()
	if result.Deleted != nil {
		deleted.Set("ids", result.Deleted)
	} else {
		// a skipped step's output is the workflow data proxy upstream
		proxy := dal.NewRecord()
		proxy.Set("__type", "Symbol(WorkflowWorkflowData)")
		deleted.Set("ids", proxy)
	}
	deleted.Set("object", "shipping_option_rule")
	deleted.Set("deleted", true)
	out := dal.NewRecord()
	out.Set("created", created)
	out.Set("updated", updated)
	out.Set("deleted", deleted)
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{POST: POST}
