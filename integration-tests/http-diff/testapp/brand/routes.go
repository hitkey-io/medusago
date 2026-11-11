package brand

import (
	"fmt"
	"strconv"
	"strings"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	modulessdk "github.com/hitkey-io/medusago/packages/core/modules-sdk/src"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// The API routes of the test application (testapp/src/api).

func rec(kv ...any) *dal.Record {
	r := dal.NewRecord()
	for i := 0; i+1 < len(kv); i += 2 {
		r.Set(kv[i].(string), kv[i+1])
	}
	return r
}

func queryString(req *medusahttp.MedusaRequest, key string) (string, bool) {
	if req.Query == nil {
		return "", false
	}
	v, ok := req.Query.Get(key)
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

func queryJSON(req *medusahttp.MedusaRequest, key string) (*dal.Record, error) {
	s, ok := queryString(req, key)
	if !ok || s == "" {
		return nil, nil
	}
	v, err := dal.DecodeOrderedJSON([]byte(s))
	if err != nil {
		return nil, err
	}
	r, _ := v.(*dal.Record)
	return r, nil
}

func fieldsOf(req *medusahttp.MedusaRequest) []string {
	if s, ok := queryString(req, "fields"); ok {
		return strings.Split(s, ",")
	}
	return []string{"*"}
}

func service(req *medusahttp.MedusaRequest) *BrandModuleService {
	return req.Scope.Resolve("brand").(*BrandModuleService)
}

func link(req *medusahttp.MedusaRequest) *modulessdk.Link {
	return req.Scope.Resolve(common.ContainerRegistrationKeys.LINK).(*modulessdk.Link)
}

func bodyRecord(req *medusahttp.MedusaRequest) *dal.Record {
	r, _ := req.Body.(*dal.Record)
	if r == nil {
		r = dal.NewRecord()
	}
	return r
}

func numberOf(s string) float64 {
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0
	}
	return f
}

// testGraph mirrors api/admin/test-graph/route.js.
func testGraph(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	filters, err := queryJSON(req, "filters")
	if err != nil {
		return err
	}
	if filters == nil {
		filters = dal.NewRecord()
	}
	pagination := dal.NewRecord()
	if s, ok := queryString(req, "offset"); ok {
		pagination.Set("skip", numberOf(s))
	}
	if s, ok := queryString(req, "limit"); ok {
		pagination.Set("take", numberOf(s))
	}
	if order, err := queryJSON(req, "order"); err != nil {
		return err
	} else if order != nil {
		pagination.Set("order", order)
	}
	entity, _ := queryString(req, "entity")
	r, err := httputils.QueryOf(req.Scope).Graph(req.Ctx(), query.GraphInput{Entity: entity, Fields: fieldsOf(req), Filters: filters, Pagination: pagination}, joiner.Options{})
	if err != nil {
		return err
	}
	var metadata any
	if r.Metadata != nil {
		metadata = r.Metadata
	}
	return res.JSON(rec("data", r.Data, "metadata", metadata))
}

// listBrands and createBrand mirror api/admin/brands/route.js.
func listBrands(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	filters, err := queryJSON(req, "filters")
	if err != nil {
		return err
	}
	if filters == nil {
		filters = dal.NewRecord()
	}
	offset, limit := 0.0, 20.0
	if s, ok := queryString(req, "offset"); ok {
		offset = numberOf(s)
	}
	if s, ok := queryString(req, "limit"); ok {
		limit = numberOf(s)
	}
	pagination := rec("skip", offset, "take", limit)
	if order, err := queryJSON(req, "order"); err != nil {
		return err
	} else if order != nil {
		pagination.Set("order", order)
	}
	r, err := httputils.QueryOf(req.Scope).Graph(req.Ctx(), query.GraphInput{Entity: "brand", Fields: fieldsOf(req), Filters: filters, Pagination: pagination}, joiner.Options{})
	if err != nil {
		return err
	}
	get := func(k string) any {
		if r.Metadata == nil {
			return nil
		}
		v, _ := r.Metadata.Get(k)
		return v
	}
	return res.JSON(rec("brands", r.Data, "count", get("count"), "offset", get("skip"), "limit", get("take")))
}

func createBrand(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	out, err := service(req).Call(req.Ctx(), "createBrands", []any{[]any{bodyRecord(req)}})
	if err != nil {
		return err
	}
	list, _ := out.([]*dal.Record)
	var brand any
	if len(list) > 0 {
		brand = list[0]
	}
	return res.JSON(rec("brand", brand))
}

// getBrand, updateBrand and deleteBrand mirror api/admin/brands/[id]/route.js.
func getBrand(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	brand, err := service(req).Call(req.Ctx(), "retrieveBrand", []any{req.Params["id"], rec("relations", []any{"logos", "profile"})})
	if err != nil {
		return err
	}
	return res.JSON(rec("brand", brand))
}

func updateBrand(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	data := rec("id", req.Params["id"])
	body := bodyRecord(req)
	for _, k := range body.Keys() {
		v, _ := body.Get(k)
		data.Set(k, v)
	}
	brand, err := service(req).Call(req.Ctx(), "updateBrands", []any{data})
	if err != nil {
		return err
	}
	return res.JSON(rec("brand", brand))
}

func deleteBrand(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	cascaded, err := service(req).Call(req.Ctx(), "softDeleteBrands", []any{[]any{req.Params["id"]}, rec("returnLinkableKeys", true)})
	if err != nil {
		return err
	}
	if r, ok := cascaded.(*dal.Record); !ok || r == nil {
		cascaded = nil
	}
	return res.JSON(rec("id", req.Params["id"], "deleted", true, "cascaded", cascaded))
}

func stringList(v any) []any {
	list, _ := v.([]any)
	return list
}

// brandProducts mirrors api/admin/brands/[id]/products/route.js.
func brandProducts(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body := bodyRecord(req)
	build := func(key string) []modulessdk.LinkDefinition {
		v, _ := body.Get(key)
		var out []modulessdk.LinkDefinition
		for _, id := range stringList(v) {
			out = append(out, rec("product", rec("product_id", id), "brand", rec("brand_id", req.Params["id"])))
		}
		return out
	}
	add, remove := build("add"), build("remove")
	created, dismissed := []*dal.Record{}, []*dal.Record{}
	var err error
	if len(add) > 0 {
		if created, err = link(req).Create(req.Ctx(), add); err != nil {
			return err
		}
	}
	if len(remove) > 0 {
		if dismissed, err = link(req).Dismiss(req.Ctx(), remove); err != nil {
			return err
		}
	}
	return res.JSON(rec("created", created, "dismissed", dismissed))
}

// brandSalesChannels mirrors api/admin/brands/[id]/sales-channels/route.js.
func brandSalesChannels(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	v, _ := bodyRecord(req).Get("links")
	var defs []modulessdk.LinkDefinition
	for _, item := range stringList(v) {
		l, ok := item.(*dal.Record)
		if !ok {
			return fmt.Errorf("invalid link")
		}
		scID, _ := l.Get("sales_channel_id")
		def := rec("brand", rec("brand_id", req.Params["id"]), "sales_channel", rec("sales_channel_id", scID))
		if data, ok := l.Get("data"); ok && data != nil {
			def.Set("data", data)
		}
		defs = append(defs, def)
	}
	created, err := link(req).Create(req.Ctx(), defs)
	if err != nil {
		return err
	}
	return res.JSON(rec("created", created))
}

// brandStats mirrors api/admin/brand-stats/route.js.
func listBrandStats(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	stats, err := service(req).Call(req.Ctx(), "listBrandStats", []any{dal.NewRecord(), rec("order", rec("brand_id", "ASC", "period", "ASC"))})
	if err != nil {
		return err
	}
	return res.JSON(rec("stats", stats))
}

func upsertBrandStats(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	v, _ := bodyRecord(req).Get("stats")
	var data []*dal.Record
	for _, item := range stringList(v) {
		if r, ok := item.(*dal.Record); ok {
			data = append(data, r)
		}
	}
	stats, err := service(req).UpsertBrandStats(req.Ctx(), data)
	if err != nil {
		return err
	}
	return res.JSON(rec("stats", stats))
}

// Routes are the route files of testapp/src/api.
var Routes = []medusahttp.RouteFile{
	{Path: "/admin/test-graph", Route: medusahttp.Route{GET: testGraph}},
	{Path: "/admin/brands", Route: medusahttp.Route{GET: listBrands, POST: createBrand}},
	{Path: "/admin/brands/:id", Route: medusahttp.Route{GET: getBrand, POST: updateBrand, DELETE: deleteBrand}},
	{Path: "/admin/brands/:id/products", Route: medusahttp.Route{POST: brandProducts}},
	{Path: "/admin/brands/:id/sales-channels", Route: medusahttp.Route{POST: brandSalesChannels}},
	{Path: "/admin/brand-stats", Route: medusahttp.Route{GET: listBrandStats, POST: upsertBrandStats}},
}
