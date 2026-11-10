package search

import (
	"context"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

// searchIndexLister is the part of the Search Module the route uses.
type searchIndexLister interface {
	ListIndexes(ctx context.Context) ([]*dal.Record, error)
}

func numberOr(v any, fallback float64) float64 {
	if f, ok := v.(float64); ok {
		return f
	}
	return fallback
}

// GET mirrors GET: per entity, the Search Module when that entity has an
// index, otherwise query.graph with the free-text `q`. The default
// application defines no search index, so every entity goes through the
// graph fallback (RBAC filtering is Enterprise: every entity is readable).
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	searchModule, _ := req.Scope.Resolve(utilsmodulessdk.SEARCH).(searchIndexLister)

	var q *string
	var entity []string
	if v, ok := req.ValidatedQuery.Get("q"); ok {
		if s, ok := v.(string); ok {
			q = &s
		}
	}
	if v, ok := req.ValidatedQuery.Get("entity"); ok {
		if arr, ok := v.([]any); ok {
			for _, e := range arr {
				s, _ := e.(string)
				entity = append(entity, s)
			}
		}
	}
	var skip, take float64 = 0, 20
	if p := req.QueryConfig.Pagination; p != nil {
		skip = numberOr(get(p, "skip"), 0)
		take = numberOr(get(p, "take"), 20)
	}

	indexEntities := map[string]string{}
	var indexNames []string
	if searchModule != nil {
		indexes, err := searchModule.ListIndexes(req.Ctx())
		if err != nil {
			return err
		}
		for _, idx := range indexes {
			name, _ := get(idx, "name").(string)
			ent, _ := get(idx, "entity").(string)
			if _, ok := indexEntities[name]; !ok {
				indexNames = append(indexNames, name)
			}
			indexEntities[name] = ent
		}
	}
	requested := entity
	if len(requested) == 0 {
		requested = indexNames
	}

	if searchModule == nil || len(requested) == 0 {
		graphEntities := entity
		if len(graphEntities) == 0 {
			graphEntities = DefaultAdminSearchEntityNames
		}
		results := []any{}
		if len(graphEntities) > 0 {
			r, err := searchWithGraphFallback(req.Ctx(), req.Scope, fallbackSearchInput{q: q, entity: graphEntities, skip: skip, take: take})
			if err != nil {
				return err
			}
			results = r
		}
		out := dal.NewRecord()
		out.Set("results", results)
		return res.JSON(out)
	}

	// Indexed entities would go through query.search; no index definition
	// is ported, so the requested names all take the graph fallback.
	var graphEntities []string
	for _, name := range requested {
		if _, indexed := indexEntities[name]; !indexed {
			graphEntities = append(graphEntities, name)
		}
	}
	byEntity := map[string]any{}
	if len(graphEntities) > 0 {
		groups, err := searchWithGraphFallback(req.Ctx(), req.Scope, fallbackSearchInput{q: q, entity: graphEntities, skip: skip, take: take})
		if err != nil {
			return err
		}
		for _, g := range groups {
			name, _ := get(g.(*dal.Record), "entity").(string)
			byEntity[name] = g
		}
	}
	results := make([]any, 0, len(requested))
	for _, name := range requested {
		results = append(results, byEntity[name])
	}
	out := dal.NewRecord()
	out.Set("results", results)
	return res.JSON(out)
}

func get(r *dal.Record, key string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(key)
	return v
}

// Route is route.ts.
var Route = medusahttp.Route{GET: GET}
