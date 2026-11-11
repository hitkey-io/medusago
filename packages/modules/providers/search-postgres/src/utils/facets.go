package utils

import (
	"fmt"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	searchutils "github.com/hitkey-io/medusago/packages/core/utils/src/search"
)

// FacetRange is one range of a normalized range facet request.
type FacetRange struct {
	Key  string
	From *float64
	To   *float64
}

// NormalizedFacetRequest mirrors NormalizedFacetRequest.
type NormalizedFacetRequest struct {
	Field string
	Type  string // value | range | stats
	// Limit is nil when undefined.
	Limit  *float64
	Sort   string
	Query  *string
	Ranges []FacetRange
}

func rangeBound(v any, present bool) *float64 {
	if !present {
		return nil
	}
	var f float64
	if n, ok := v.(float64); ok {
		f = n
	} else {
		f = searchutils.JSNumber(v)
	}
	return &f
}

func boundKey(v any, present bool) string {
	if !present || v == nil {
		return "*"
	}
	return searchutils.JSString(v)
}

// NormalizeFacetRequests mirrors normalizeFacetRequests (requests are
// strings or facet request objects).
func NormalizeFacetRequests(requests []any, plan *IndexPlan) ([]NormalizedFacetRequest, error) {
	out := []NormalizedFacetRequest{}
	for _, request := range requests {
		if s, ok := request.(string); ok {
			planned := plan.Field(s)
			if planned == nil || !IsFacetable(planned.Field) {
				return nil, fail(fmt.Sprintf(`Field "%s" is not facetable on the postgres search provider`, s))
			}
			out = append(out, NormalizedFacetRequest{Field: s, Type: "value"})
			continue
		}
		r, _ := request.(*dal.Record)
		field, _ := recordGet(r, "field").(string)
		planned := plan.Field(field)
		if planned == nil || !IsFacetable(planned.Field) {
			return nil, fail(fmt.Sprintf(`Field "%s" is not facetable on the postgres search provider`, field))
		}
		typ, _ := recordGet(r, "type").(string)
		switch typ {
		case "stats":
			if planned.Kind != "number" {
				return nil, fail(fmt.Sprintf(`Stats facets require a numeric field ("%s")`, field))
			}
			out = append(out, NormalizedFacetRequest{Field: field, Type: "stats"})
			continue
		case "range":
			if planned.Kind != "number" {
				return nil, fail(fmt.Sprintf(`Range facets require a numeric field ("%s")`, field))
			}
			ranges := []FacetRange{}
			rawRanges, _ := recordGet(r, "ranges").([]any)
			for _, rr := range rawRanges {
				rng, _ := rr.(*dal.Record)
				from, hasFrom := rng.Get("from")
				to, hasTo := rng.Get("to")
				hasFrom = hasFrom && from != nil
				hasTo = hasTo && to != nil
				key, hasKey := rng.Get("key")
				k, _ := key.(string)
				if !hasKey || key == nil {
					k = boundKey(from, hasFrom) + "-" + boundKey(to, hasTo)
				}
				ranges = append(ranges, FacetRange{Key: k, From: rangeBound(from, hasFrom), To: rangeBound(to, hasTo)})
			}
			out = append(out, NormalizedFacetRequest{Field: field, Type: "range", Ranges: ranges})
			continue
		}
		req := NormalizedFacetRequest{Field: field, Type: "value"}
		if l, ok := recordGet(r, "limit").(float64); ok {
			req.Limit = &l
		}
		req.Sort, _ = recordGet(r, "sort").(string)
		if q, ok := recordGet(r, "query").(string); ok {
			req.Query = &q
		}
		out = append(out, req)
	}
	return out, nil
}

// BuildFacetQuery mirrors buildFacetQuery.
func BuildFacetQuery(table string, where *SqlFragment, request NormalizedFacetRequest, plan *IndexPlan) SqlFragment {
	planned := plan.Field(request.Field)
	var whereSQL string
	var whereParams []any
	if where != nil {
		whereSQL, whereParams = where.SQL, where.Params
	}
	params := append([]any{}, whereParams...)
	whereClause := ""
	if whereSQL != "" {
		whereClause = "WHERE " + whereSQL
	}

	if request.Type == "stats" {
		expr := fmt.Sprintf("(%s)::float8", JsonbPath(request.Field, true))
		return SqlFragment{SQL: fmt.Sprintf(`
        SELECT
          MIN(%[1]s) AS min,
          MAX(%[1]s) AS max,
          AVG(%[1]s) AS avg,
          SUM(%[1]s) AS sum,
          COUNT(*)::int AS count
        FROM "%[2]s"
        %[3]s
      `, expr, table, whereClause), Params: params}
	}

	if request.Type == "range" {
		expr := fmt.Sprintf("(%s)::float8", JsonbPath(request.Field, true))
		rangeParams := []any{}
		selects := make([]string, len(request.Ranges))
		for i, rng := range request.Ranges {
			var conditions []string
			if rng.From != nil {
				rangeParams = append(rangeParams, *rng.From)
				conditions = append(conditions, expr+" >= ?")
			}
			if rng.To != nil {
				rangeParams = append(rangeParams, *rng.To)
				conditions = append(conditions, expr+" < ?")
			}
			predicate := "TRUE"
			if len(conditions) > 0 {
				predicate = strings.Join(conditions, " AND ")
			}
			selects[i] = fmt.Sprintf("COUNT(*) FILTER (WHERE %s)::int AS r%d", predicate, i)
		}
		return SqlFragment{
			SQL:    fmt.Sprintf(`SELECT %s FROM "%s" %s`, strings.Join(selects, ",\n"), table, whereClause),
			Params: append(rangeParams, whereParams...),
		}
	}

	valueExpr := JsonbPath(request.Field, true)
	if planned.IsArray {
		valueExpr = fmt.Sprintf("jsonb_array_elements_text(%s)", JsonbPath(request.Field, false))
	}
	var limit any = float64(10)
	if request.Limit != nil {
		limit = *request.Limit
	}
	params = append(params, limit)
	valueFilter := ""
	if request.Query != nil && *request.Query != "" {
		params = append(params[:len(params)-1], *request.Query+"%", limit)
		if planned.IsArray {
			valueFilter = "AND value LIKE ?"
		} else {
			valueFilter = "AND " + valueExpr + " LIKE ?"
		}
	}
	order := "count DESC, value ASC"
	if request.Sort == "alpha" {
		order = "value ASC"
	}
	if planned.IsArray {
		return SqlFragment{SQL: fmt.Sprintf(`
        SELECT value, COUNT(*)::int AS count
        FROM (
          SELECT %s AS value
          FROM "%s"
          %s
        ) AS exploded
        WHERE value IS NOT NULL
        %s
        GROUP BY value
        ORDER BY %s
        LIMIT ?
      `, valueExpr, table, whereClause, valueFilter, order), Params: params}
	}
	nullGuard := valueExpr + " IS NOT NULL"
	facetWhere := "WHERE " + nullGuard
	if whereSQL != "" {
		facetWhere = "WHERE " + whereSQL + " AND " + nullGuard
	}
	return SqlFragment{SQL: fmt.Sprintf(`
      SELECT %s AS value, COUNT(*)::int AS count
      FROM "%s"
      %s
      %s
      GROUP BY 1
      ORDER BY %s
      LIMIT ?
    `, valueExpr, table, facetWhere, valueFilter, order), Params: params}
}

func numberOf(v any) float64 {
	if v == nil {
		return 0
	}
	return searchutils.JSNumber(v)
}

// MapFacetResult mirrors mapFacetResult.
func MapFacetResult(request NormalizedFacetRequest, rows []dal.Row) *dal.Record {
	out := dal.NewRecord()
	row := dal.Row{}
	if len(rows) > 0 {
		row = rows[0]
	}
	switch request.Type {
	case "stats":
		out.Set("type", "stats")
		out.Set("min", numberOf(row["min"]))
		out.Set("max", numberOf(row["max"]))
		if row["avg"] != nil {
			out.Set("avg", numberOf(row["avg"]))
		}
		if row["sum"] != nil {
			out.Set("sum", numberOf(row["sum"]))
		}
		out.Set("count", numberOf(row["count"]))
		return out
	case "range":
		out.Set("type", "range")
		ranges := make([]any, len(request.Ranges))
		for i, rng := range request.Ranges {
			r := dal.NewRecord()
			r.Set("key", rng.Key)
			if rng.From != nil {
				r.Set("from", *rng.From)
			}
			if rng.To != nil {
				r.Set("to", *rng.To)
			}
			r.Set("count", numberOf(row[fmt.Sprintf("r%d", i)]))
			ranges[i] = r
		}
		out.Set("ranges", ranges)
		return out
	}
	out.Set("type", "value")
	values := make([]any, len(rows))
	for i, rw := range rows {
		v := dal.NewRecord()
		v.Set("value", searchutils.JSString(rw["value"]))
		v.Set("count", numberOf(rw["count"]))
		values[i] = v
	}
	out.Set("values", values)
	return out
}
