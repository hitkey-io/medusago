// Package batch mirrors api/admin/translations/batch/route.ts.
package batch

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/translation/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/translations"
)

// BatchBody reads { create, update, delete } of a validated batch body.
func BatchBody(body *dal.Record) workflows.BatchInput {
	in := workflows.BatchInput{Create: []*dal.Record{}, Update: []*dal.Record{}, Delete: []any{}}
	records := func(key string) []*dal.Record {
		out := []*dal.Record{}
		v, _ := body.Get(key)
		list, _ := v.([]any)
		for _, x := range list {
			if r, ok := x.(*dal.Record); ok {
				out = append(out, r)
			}
		}
		return out
	}
	if body == nil {
		return in
	}
	in.Create = records("create")
	in.Update = records("update")
	if v, ok := body.Get("delete"); ok {
		if list, isList := v.([]any); isList {
			in.Delete = list
		}
	}
	return in
}

func idsOf(rs []*dal.Record) map[any]bool {
	out := map[any]bool{}
	for _, r := range rs {
		id, _ := r.Get("id")
		out[id] = true
	}
	return out
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	in := BatchBody(body)
	result, err := workflows.BatchTranslationsWorkflow.Run(req.Ctx(), req.Scope, in)
	if err != nil {
		return err
	}
	var ids []any
	seen := map[any]bool{}
	for _, r := range append(append([]*dal.Record{}, result.Created...), result.Updated...) {
		id, _ := r.Get("id")
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	if ids == nil {
		ids = []any{}
	}
	filters := dal.NewRecord()
	filters.Set("id", ids)
	data, err := httputils.QueryOf(req.Scope).Graph(req.Ctx(), query.GraphInput{Entity: "translation", Fields: translations.DefaultAdminTranslationFields, Filters: filters}, joiner.Options{})
	if err != nil {
		return err
	}
	createdIDs, updatedIDs := idsOf(result.Created), idsOf(result.Updated)
	created, updated := []any{}, []any{}
	for _, t := range data.Data {
		r, _ := t.(*dal.Record)
		if r == nil {
			continue
		}
		id, _ := r.Get("id")
		if createdIDs[id] {
			created = append(created, r)
		}
		if updatedIDs[id] {
			updated = append(updated, r)
		}
	}
	deleted := dal.NewRecord()
	deleted.Set("ids", in.Delete)
	deleted.Set("object", "translation")
	deleted.Set("deleted", true)
	out := dal.NewRecord()
	out.Set("created", created)
	out.Set("updated", updated)
	out.Set("deleted", deleted)
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{POST: POST}
