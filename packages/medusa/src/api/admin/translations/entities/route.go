// Package entities mirrors api/admin/translations/entities/route.ts (no
// defineFileConfig: the route is registered whatever the flag).
package entities

import (
	"fmt"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/caching"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/translations/statistics"
)

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	q := httputils.QueryOf(req.Scope)
	typ, _ := req.ValidatedQuery.Get("type")
	id, hasID := req.ValidatedQuery.Get("id")

	settingsFilter := dal.NewRecord()
	settingsFilter.Set("entity_type", typ)
	settings, err := q.Graph(req.Ctx(), query.GraphInput{Entity: "translation_settings", Fields: []string{"*"}, Filters: settingsFilter}, joiner.Options{Cache: caching.Enabled()})
	if err != nil {
		return err
	}
	fields := []string{"id"}
	if len(settings.Data) > 0 {
		if s, ok := settings.Data[0].(*dal.Record); ok {
			f, _ := s.Get("fields")
			if list, ok := f.([]any); ok {
				for _, x := range list {
					fields = append(fields, fmt.Sprint(x))
				}
			}
		}
	}

	filters := dal.NewRecord()
	if hasID && id != nil {
		filters.Set("id", id)
	}
	var data []any
	metadata := dal.NewRecord()
	entities, err := q.Graph(req.Ctx(), query.GraphInput{Entity: fmt.Sprint(typ), Fields: fields, Filters: filters, Pagination: req.QueryConfig.Pagination}, joiner.Options{Cache: caching.Enabled()})
	if err != nil {
		if !statistics.IsServiceNotFound(err) {
			return err
		}
		data = []any{}
		metadata.Set("count", float64(0))
		metadata.Set("skip", float64(0))
		metadata.Set("take", float64(0))
	} else {
		data = entities.Data
		if entities.Metadata != nil {
			metadata = entities.Metadata
		}
	}

	if len(data) > 0 {
		refs := make([]any, 0, len(data))
		for _, e := range data {
			if r, ok := e.(*dal.Record); ok {
				v, _ := r.Get("id")
				refs = append(refs, v)
			}
		}
		tf := dal.NewRecord()
		tf.Set("reference_id", refs)
		translations, err := q.Graph(req.Ctx(), query.GraphInput{Entity: "translations", Fields: []string{"*"}, Filters: tf}, joiner.Options{})
		if err != nil {
			return err
		}
		for _, e := range data {
			r, ok := e.(*dal.Record)
			if !ok {
				continue
			}
			eid, _ := r.Get("id")
			matched := []any{}
			for _, t := range translations.Data {
				if tr, ok := t.(*dal.Record); ok {
					if ref, _ := tr.Get("reference_id"); ref == eid {
						matched = append(matched, tr)
					}
				}
			}
			r.Set("translations", matched)
		}
	}

	out := dal.NewRecord()
	out.Set("data", data)
	count, skip, take := any(float64(0)), any(float64(0)), any(float64(0))
	if v, ok := metadata.Get("count"); ok && v != nil {
		count = v
	}
	if v, ok := metadata.Get("skip"); ok && v != nil {
		skip = v
	}
	if v, ok := metadata.Get("take"); ok && v != nil {
		take = v
	}
	out.Set("count", count)
	out.Set("offset", skip)
	out.Set("limit", take)
	return res.JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET}
