package search

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/framework/src/http/middlewares"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	searchtypes "github.com/hitkey-io/medusago/packages/core/types/src/search"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

const (
	productEntity     = "product"
	statusField       = "status"
	salesChannelField = "sales_channel_ids"
	productPublished  = "published"
)

// Middlewares mirrors storeSearchRoutesMiddlewares.
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"POST"}, Matcher: "/store/search", Middlewares: []medusahttp.MiddlewareFunction{
		middlewares.Authenticate([]string{"customer"}, []string{"session", "bearer"}, medusahttp.AuthenticateOptions{AllowUnauthenticated: true}),
		httputils.ValidateAndTransformBody(httputils.Schema(StoreSearch)),
	}},
}

// searchModule is the part of the Search Module the route uses.
type searchModule interface {
	GetIndex(index string) (*searchtypes.ResolvedDefinition, error)
	ListRetrievableFields(index string) ([]string, error)
}

// unscopedProductIndexes mirrors unscopedProductIndexes: the warning is
// logged once per index.
var unscopedProductIndexes sync.Map

func notFound(entity string) error {
	return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf(`No search index named "%s"`, entity))
}

func get(r *dal.Record, key string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(key)
	return v
}

func logger(req *medusahttp.MedusaRequest) *slog.Logger {
	if l, ok := req.Scope.Resolve(common.ContainerRegistrationKeys.LOGGER).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}

type plan struct {
	productFilters *dal.Record
	filters        *dal.Record
}

// POST mirrors POST: the engine's own results for the queries, limited to
// the indexes configureStoreSearch opted in.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	module, _ := req.Scope.Resolve(utilsmodulessdk.SEARCH).(searchModule)
	if module == nil {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, "The Search Module is not installed, so nothing can be searched.")
	}
	allowed := map[string]any{}
	if config := middlewares.GetStoreSearchConfig(req); config != nil {
		allowed = config.AllowedIndexes
	}
	body, _ := req.ValidatedBody.(*dal.Record)
	var queries []*dal.Record
	if body != nil && body.Has("queries") {
		arr, _ := get(body, "queries").([]any)
		for _, q := range arr {
			r, _ := q.(*dal.Record)
			queries = append(queries, r)
		}
	} else {
		queries = []*dal.Record{body}
	}

	plans := make([]plan, len(queries))
	for i, q := range queries {
		entity, _ := get(q, "entity").(string)
		config, ok := allowed[entity]
		if !ok || config == nil || config == false {
			logger(req).Warn(fmt.Sprintf("Search index \"%s\" is not exposed on /store/search. Opt it in with the `configureStoreSearch` middleware.", entity))
			return notFound(entity)
		}
		index, err := module.GetIndex(entity)
		if err != nil {
			return notFound(entity)
		}
		if fields, ok := get(q, "fields").([]any); ok {
			retrievableFields, err := module.ListRetrievableFields(entity)
			if err != nil {
				return err
			}
			retrievable := map[string]bool{}
			for _, f := range retrievableFields {
				retrievable[f] = true
			}
			var unknown []string
			for _, f := range fields {
				s, _ := f.(string)
				if !retrievable[s] {
					unknown = append(unknown, s)
				}
			}
			if len(unknown) > 0 {
				return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf(`Search index "%s" can't return the fields: %s`, entity, strings.Join(unknown, ", ")))
			}
		}
		productFilters, err := buildProductFilters(index, req)
		if err != nil {
			return err
		}
		p := plan{productFilters: productFilters}
		if c, ok := config.(*middlewares.StoreSearchIndexConfig); ok && c.Filters != nil {
			f, err := c.Filters(req)
			if err != nil {
				return err
			}
			p.filters = f
		}
		plans[i] = p
	}

	searchQueries := make([]*dal.Record, len(queries))
	for i, q := range queries {
		sq := dal.NewRecord()
		for _, k := range q.Keys() {
			v, _ := q.Get(k)
			sq.Set(k, v)
		}
		queryFilters, _ := get(q, "filters").(*dal.Record)
		merged := mergeSearchFilters(plans[i].productFilters, plans[i].filters, queryFilters)
		if merged != nil {
			sq.Set("filters", merged)
		} else {
			sq.Delete("filters")
		}
		searchQueries[i] = sq
	}
	qr, _ := req.Scope.Resolve(common.ContainerRegistrationKeys.QUERY).(*query.Query)
	results, err := qr.Search(req.Ctx(), searchQueries, joiner.Options{})
	if err != nil {
		return err
	}
	out := make([]any, len(results))
	for i, r := range results {
		out[i] = r.SearchResult
	}
	resp := dal.NewRecord()
	resp.Set("results", out)
	return res.JSON(resp)
}

func isTrue(b *bool) bool { return b != nil && *b }

// buildProductFilters mirrors buildProductFilters: a product index is
// narrowed to published products of the key's sales channels.
func buildProductFilters(index *searchtypes.ResolvedDefinition, req *medusahttp.MedusaRequest) (*dal.Record, error) {
	if index.Entity != productEntity {
		return nil, nil
	}
	filters := dal.NewRecord()
	var missing []string
	if f := index.Fields.Get(statusField); f != nil && isTrue(f.Filterable) {
		filters.Set(statusField, productPublished)
	} else {
		missing = append(missing, statusField)
	}
	if f := index.Fields.Get(salesChannelField); f != nil && isTrue(f.Filterable) {
		ids, _ := get(req.PublishableKeyContext, "sales_channel_ids").([]any)
		if len(ids) == 0 {
			return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Publishable key needs to have a sales channel configured")
		}
		filters.Set(salesChannelField, ids)
	} else {
		missing = append(missing, salesChannelField)
	}
	if len(missing) > 0 {
		if _, logged := unscopedProductIndexes.LoadOrStore(index.Name, true); !logged {
			quoted := make([]string, len(missing))
			for i, m := range missing {
				quoted[i] = `"` + m + `"`
			}
			logger(req).Warn(fmt.Sprintf("Search index \"%s\" holds products but declares no filterable %s, so /store/search cannot narrow it there. Add the field to the index, or scope it through the `filters` of `configureStoreSearch`.", index.Name, strings.Join(quoted, " or ")))
		}
	}
	if filters.Len() == 0 {
		return nil, nil
	}
	return filters, nil
}

// mergeSearchFilters mirrors mergeSearchFilters: the filters ANDed, `q`
// hoisted to the top level.
func mergeSearchFilters(filters ...*dal.Record) *dal.Record {
	var present []*dal.Record
	for _, f := range filters {
		if f != nil && f.Len() > 0 {
			present = append(present, f)
		}
	}
	if len(present) < 2 {
		if len(present) == 0 {
			return nil
		}
		return present[0]
	}
	var q any
	hasQ := false
	var branches []any
	for _, filter := range present {
		rest := dal.NewRecord()
		for _, k := range filter.Keys() {
			v, _ := filter.Get(k)
			if k == "q" {
				if v != nil {
					q, hasQ = v, true
				}
				continue
			}
			rest.Set(k, v)
		}
		if rest.Len() > 0 {
			branches = append(branches, rest)
		}
	}
	out := dal.NewRecord()
	if hasQ && isPresent(q) {
		out.Set("q", q)
	}
	if len(branches) > 1 {
		out.Set("$and", branches)
	} else if len(branches) == 1 {
		only := branches[0].(*dal.Record)
		for _, k := range only.Keys() {
			v, _ := only.Get(k)
			out.Set(k, v)
		}
	}
	return out
}

// isPresent mirrors isPresent for the free-text value.
func isPresent(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case string:
		return t != ""
	}
	return true
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
