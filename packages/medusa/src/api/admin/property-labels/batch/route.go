// Package batch mirrors api/admin/property-labels/batch.
package batch

import (
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/settings/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	propertylabels "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/property-labels"
)

func records(v any, keys ...string) []*dal.Record {
	list, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]*dal.Record, 0, len(list))
	for _, x := range list {
		r, _ := x.(*dal.Record)
		if r == nil {
			r = dal.NewRecord()
		}
		out = append(out, propertylabels.Pick(r, keys...))
	}
	return out
}

func idsOf(rows []*dal.Record) []string {
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		id, _ := r.Get("id")
		out = append(out, fmt.Sprint(id))
	}
	return out
}

// POST mirrors POST: the created and updated labels are refetched, the
// deleted ids answered as deletion objects.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	if body == nil {
		body = dal.NewRecord()
	}
	in := workflows.BatchPropertyLabelsWorkflowInput{}
	if v, _ := body.Get("create"); v != nil {
		in.Create = records(v, "entity", "property", "label", "description")
	}
	if v, _ := body.Get("update"); v != nil {
		in.Update = records(v, "id", "label", "description")
	}
	if v, _ := body.Get("delete"); v != nil {
		if list, ok := v.([]any); ok {
			in.Delete = list
		}
	}
	result, err := workflows.BatchPropertyLabelsWorkflow.Run(req.Ctx(), req.Scope, in)
	if err != nil {
		return err
	}
	createdIDs := idsOf(result.Created)
	updatedIDs := idsOf(result.Updated)
	created := map[string]bool{}
	updated := map[string]bool{}
	seen := map[string]bool{}
	ids := []any{}
	for _, id := range createdIDs {
		created[id] = true
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	for _, id := range updatedIDs {
		updated[id] = true
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	f := dal.NewRecord()
	f.Set("id", ids)
	r, err := propertylabels.Graph(req.Ctx(), req, f, nil)
	if err != nil {
		return err
	}
	createdLabels, updatedLabels := []any{}, []any{}
	for _, row := range r.Data {
		rec, _ := row.(*dal.Record)
		if rec == nil {
			continue
		}
		id, _ := rec.Get("id")
		key := fmt.Sprint(id)
		if created[key] {
			createdLabels = append(createdLabels, row)
		}
		if updated[key] {
			updatedLabels = append(updatedLabels, row)
		}
	}
	deleted := []any{}
	for _, id := range result.Deleted {
		d := dal.NewRecord()
		d.Set("id", id)
		d.Set("object", "property_label")
		d.Set("deleted", true)
		deleted = append(deleted, d)
	}
	out := dal.NewRecord()
	out.Set("created", createdLabels)
	out.Set("updated", updatedLabels)
	out.Set("deleted", deleted)
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{POST: POST}
