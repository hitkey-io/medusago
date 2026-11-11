package httpdiff

import (
	"fmt"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	indexservices "github.com/hitkey-io/medusago/packages/modules/index/src/services"
	indexutils "github.com/hitkey-io/medusago/packages/modules/index/src/utils"
)

// indexQueryRoutes is the test-only /index-query route of the reference
// application (index-query-route.js, copied by record.sh): POST runs
// query.index(body), or the index module's query(body) with
// "mode": "module".
func indexQueryRoutes() []medusahttp.RouteFile {
	return []medusahttp.RouteFile{{Path: "/index-query", Route: medusahttp.Route{POST: indexQueryRoute}}}
}

func indexQueryRoute(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.Body.(*dal.Record)
	if body == nil {
		body = dal.NewRecord()
	}
	record := func(key string) *dal.Record {
		v, _ := body.Get(key)
		r, _ := v.(*dal.Record)
		return r
	}
	var fields []string
	if v, _ := body.Get("fields"); v != nil {
		list, _ := v.([]any)
		for _, f := range list {
			fields = append(fields, fmt.Sprint(f))
		}
	}
	out := dal.NewRecord()
	if mode, _ := body.Get("mode"); mode == "module" {
		index, ok := req.Scope.Resolve(utilsmodulessdk.INDEX).(*indexservices.IndexModuleService)
		if !ok {
			return fmt.Errorf("the index module is not loaded")
		}
		idsOnly, _ := body.Get("idsOnly")
		r, err := index.Query(req.Ctx(), indexutils.IndexQueryConfig{
			Fields:      fields,
			Filters:     record("filters"),
			JoinFilters: record("joinFilters"),
			Pagination:  record("pagination"),
			IdsOnly:     idsOnly == true,
		})
		if err != nil {
			return err
		}
		out.Set("data", nonNil(r.Data))
		if r.Metadata != nil {
			out.Set("metadata", r.Metadata)
		}
		return res.JSON(out)
	}
	entity, _ := body.Get("entity")
	r, err := httputils.QueryOf(req.Scope).Index(req.Ctx(), query.GraphInput{
		Entity:     fmt.Sprint(entity),
		Fields:     fields,
		Filters:    record("filters"),
		Pagination: record("pagination"),
	}, record("joinFilters"), joiner.Options{})
	if err != nil {
		return err
	}
	out.Set("data", nonNil(r.Data))
	if r.Metadata != nil {
		out.Set("metadata", r.Metadata)
	}
	return res.JSON(out)
}

func nonNil(v []any) []any {
	if v == nil {
		return []any{}
	}
	return v
}
