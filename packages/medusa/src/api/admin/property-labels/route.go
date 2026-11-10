// Package propertylabels mirrors api/admin/property-labels.
package propertylabels

import (
	"context"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/settings/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/settings/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// Graph runs query.graph({ entity: "property_label", fields, filters,
// pagination }).
func Graph(ctx context.Context, req *medusahttp.MedusaRequest, filters, pagination *dal.Record) (*query.GraphResult, error) {
	res, err := httputils.QueryOf(req.Scope).Graph(ctx, query.GraphInput{Entity: "property_label", Fields: req.QueryConfig.Fields, Filters: filters, Pagination: pagination}, joiner.Options{})
	if err != nil {
		return nil, err
	}
	if res.Metadata == nil {
		res.Metadata = dal.NewRecord()
	}
	return res, nil
}

// ByID returns the property label with id (nil when missing).
func ByID(req *medusahttp.MedusaRequest, id any) (any, error) {
	f := dal.NewRecord()
	f.Set("id", id)
	r, err := Graph(req.Ctx(), req, f, nil)
	if err != nil || len(r.Data) == 0 {
		return nil, err
	}
	return r.Data[0], nil
}

// Pick copies the keys of r whose values are defined and not null
// (`value ?? undefined` for the optional ones).
func Pick(r *dal.Record, keys ...string) *dal.Record {
	out := dal.NewRecord()
	for _, k := range keys {
		if v, ok := r.Get(k); ok && v != nil {
			out.Set(k, v)
		}
	}
	return out
}

func orDefault(v any, def any) any {
	if v == nil {
		return def
	}
	return v
}

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	r, err := Graph(req.Ctx(), req, req.FilterableFields, req.QueryConfig.Pagination)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("property_labels", r.Data)
	out.Set("count", orDefault(httputils.MetaValue(r.Metadata, "count"), 0.0))
	out.Set("offset", orDefault(httputils.MetaValue(r.Metadata, "skip"), 0.0))
	out.Set("limit", orDefault(httputils.MetaValue(r.Metadata, "take"), 0.0))
	return res.JSON(out)
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	label := Pick(body, "entity", "property", "label", "description")
	result, err := workflows.CreatePropertyLabelsWorkflow.Run(req.Ctx(), req.Scope, steps.CreatePropertyLabelsStepInput{PropertyLabels: []*dal.Record{label}})
	if err != nil {
		return err
	}
	var id any
	if len(result) > 0 {
		id, _ = result[0].Get("id")
	}
	pl, err := ByID(req, id)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	if pl != nil {
		out.Set("property_label", pl)
	}
	return res.Status(201).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET, POST: POST}
