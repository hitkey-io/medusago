// Package utils mirrors api/store/utils.
package utils

import (
	"context"
	"fmt"
	"strings"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

func normalizeField(f string) string {
	if strings.HasPrefix(f, "*") {
		return f[1:]
	}
	return strings.TrimSuffix(f, ".*")
}

// BuildAllowedFields mirrors buildAllowedFields.
func BuildAllowedFields(lists ...[]string) []string {
	var out []string
	seen := map[string]bool{}
	for _, l := range lists {
		for _, f := range l {
			n := normalizeField(f)
			if !seen[n] {
				seen[n] = true
				out = append(out, n)
			}
		}
	}
	return out
}

// PrefixAllowedFields mirrors prefixAllowedFields.
func PrefixAllowedFields(prefix string, lists ...[]string) []string {
	fields := BuildAllowedFields(lists...)
	out := make([]string, len(fields))
	for i, f := range fields {
		out[i] = prefix + "." + f
	}
	return out
}

// PublishedProductsContext mirrors publishedProductsContext.
func PublishedProductsContext() *dal.Record {
	published := dal.NewRecord()
	published.Set("published", true)
	v := dal.NewRecord()
	v.Set("products", published)
	return query.QueryContext(v)
}

func copyFilters(r *dal.Record) *dal.Record {
	out := dal.NewRecord()
	if r != nil {
		for _, k := range r.Keys() {
			v, _ := r.Get(k)
			out.Set(k, v)
		}
	}
	return out
}

// Graph runs query.graph with { locale } (the request locale).
func Graph(ctx context.Context, scope medusahttp.MedusaContainer, in query.GraphInput, locale string) (*query.GraphResult, error) {
	r, err := httputils.QueryOf(scope).Graph(ctx, in, joiner.Options{Locale: locale})
	if err != nil {
		return nil, err
	}
	if r.Metadata == nil {
		r.Metadata = dal.NewRecord()
	}
	return r, nil
}

// GraphListRoute builds the list routes written as query.graph({ entity,
// filters, pagination, fields, context? }); zeroMeta mirrors
// `metadata?.count ?? 0`.
func GraphListRoute(entity, key string, published, zeroMeta bool) medusahttp.RouteHandler {
	return func(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
		in := query.GraphInput{Entity: entity, Fields: req.QueryConfig.Fields, Filters: copyFilters(req.FilterableFields), Pagination: req.QueryConfig.Pagination}
		if published {
			in.Context = PublishedProductsContext()
		}
		r, err := Graph(req.Ctx(), req.Scope, in, req.Locale)
		if err != nil {
			return err
		}
		data := r.Data
		if data == nil {
			data = []any{}
		}
		out := dal.NewRecord()
		out.Set(key, data)
		for _, p := range [][2]string{{"count", "count"}, {"offset", "skip"}, {"limit", "take"}} {
			v, ok := r.Metadata.Get(p[1])
			if (!ok || v == nil) && zeroMeta {
				v, ok = float64(0), true
			}
			if ok {
				out.Set(p[0], v)
			}
		}
		return res.JSON(out)
	}
}

// GraphRetrieveRoute builds the retrieve routes written as query.graph with
// an id filter; a missing row is NOT_FOUND with notFound (a format of the
// id).
func GraphRetrieveRoute(entity, key, notFound string, published bool) medusahttp.RouteHandler {
	return func(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
		f := dal.NewRecord()
		f.Set("id", req.Params["id"])
		in := query.GraphInput{Entity: entity, Fields: req.QueryConfig.Fields, Filters: f}
		if published {
			in.Context = PublishedProductsContext()
		}
		r, err := Graph(req.Ctx(), req.Scope, in, req.Locale)
		if err != nil {
			return err
		}
		if len(r.Data) == 0 {
			return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf(notFound, req.Params["id"]))
		}
		out := dal.NewRecord()
		out.Set(key, r.Data[0])
		return res.JSON(out)
	}
}
