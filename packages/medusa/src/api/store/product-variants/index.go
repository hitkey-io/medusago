// Package productvariants mirrors api/store/product-variants.
package productvariants

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/framework/src/http/middlewares"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/store/products"
	storeutils "github.com/hitkey-io/medusago/packages/medusa/src/api/store/utils"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
	productmw "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/middlewares/products"
)

// DefaultStoreProductVariantFields mirrors defaultStoreProductVariantFields.
var DefaultStoreProductVariantFields = []string{"id", "title", "sku", "barcode", "ean", "upc", "allow_backorder", "manage_inventory", "variant_rank", "product_id", "thumbnail", "hs_code", "origin_country", "mid_code", "material", "weight", "length", "height", "width", "created_at", "updated_at", "metadata", "*options", "*images", "product.id", "product.type_id"}

// AllowedStoreProductVariantExtraFields mirrors allowedStoreProductVariantExtraFields.
var AllowedStoreProductVariantExtraFields = []string{"deleted_at", "calculated_price", "inventory_quantity", "options.id", "options.value", "images.id", "images.url", "images.rank"}

var allowed = storeutils.BuildAllowedFields(DefaultStoreProductVariantFields, AllowedStoreProductVariantExtraFields)

// Query configs.
var (
	RetrieveConfig = httputils.QueryConfig{Defaults: DefaultStoreProductVariantFields, Allowed: allowed}
	ListConfig     = httputils.QueryConfig{Defaults: DefaultStoreProductVariantFields, Allowed: allowed, DefaultLimit: 20, IsList: true}
)

var contextFields = zod.Object(
	zod.F("region_id", zod.String().Optional()),
	zod.F("country_code", zod.String().Optional()),
	zod.F("province", zod.String().Optional()),
	zod.F("cart_id", zod.String().Optional()),
	zod.F("sales_channel_id", utils.StringOrStrings().Optional()),
)

var filterFields = zod.Object(
	zod.F("q", zod.String().Optional()),
	zod.F("id", utils.StringOrStrings().Optional()),
	zod.F("sku", utils.StringOrStrings().Optional()),
	zod.F("product_id", utils.StringOrStrings().Optional()),
	zod.F("options", zod.Object(zod.F("value", zod.String().Optional()), zod.F("option_id", zod.String().Optional())).Optional()),
	zod.F("allow_backorder", commonvalidators.BooleanString().Optional()),
	zod.F("manage_inventory", commonvalidators.BooleanString().Optional()),
	zod.F("created_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("updated_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("deleted_at", utils.CreateOperatorMap(nil, nil).Optional()),
)

// StoreProductVariantParams mirrors StoreProductVariantParams.
var StoreProductVariantParams = utils.CreateSelectParams().Merge(contextFields)

// StoreProductVariantListParams mirrors StoreProductVariantListParams.
var StoreProductVariantListParams = utils.CreateFindParams(utils.FindParamsOptions{Offset: utils.Num(0), Limit: utils.Num(20)}).
	Merge(contextFields).Merge(filterFields).Merge(commonvalidators.ApplyAndAndOrOperators(filterFields))

var link = httputils.LinkFilterOptions{EntryPoint: "product_sales_channel", ResourceID: "product_id", FilterableField: "sales_channel_id", FilterByField: "product.id"}

func publishedProduct() middlewares.DefaultFilter {
	v := dal.NewRecord()
	v.Set("status", "published")
	return middlewares.DefaultFilter{Key: "product", Value: v}
}

func chain(first ...medusahttp.MiddlewareFunction) []medusahttp.MiddlewareFunction {
	return append(first,
		productmw.FilterByValidSalesChannels(),
		httputils.MaybeApplyLinkFilter(link),
		middlewares.ApplyDefaultFilters(publishedProduct()),
		productmw.NormalizeDataForContext("calculated_price"),
		productmw.SetPricingContext("calculated_price"),
		productmw.SetTaxContext("calculated_price"),
		middlewares.ClearFiltersByKey("region_id", "country_code", "province", "cart_id"),
	)
}

func authenticate() medusahttp.MiddlewareFunction {
	return middlewares.Authenticate([]string{"customer"}, []string{"session", "bearer"}, medusahttp.AuthenticateOptions{AllowUnauthenticated: true})
}

// Middlewares mirrors storeProductVariantRoutesMiddlewares.
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/store/product-variants", Middlewares: chain(authenticate(), httputils.ValidateAndTransformQuery(StoreProductVariantListParams, ListConfig))},
	{Methods: []string{"GET"}, Matcher: "/store/product-variants/:id", Middlewares: chain(authenticate(), httputils.ValidateAndTransformQuery(StoreProductVariantParams, RetrieveConfig), middlewares.ApplyParamsAsFilters(map[string]string{"id": "id"}))},
}

func get(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}

// PricingContext builds { calculated_price: QueryContext(pricingContext) }.
func PricingContext(req *medusahttp.MedusaRequest) *dal.Record {
	ctx := dal.NewRecord()
	if req.PricingContext != nil {
		ctx.Set("calculated_price", query.QueryContext(req.PricingContext))
	}
	return ctx
}

// WrapVariantsWithTaxPrices mirrors wrapVariantsWithTaxPrices.
func WrapVariantsWithTaxPrices(req *medusahttp.MedusaRequest, variants []any) error {
	lineContext := products.TaxContexts(req)
	if lineContext == nil || len(variants) == 0 {
		return nil
	}
	var items []products.TaxedVariant
	for _, x := range variants {
		v, ok := x.(*dal.Record)
		if !ok || v == nil {
			continue
		}
		if cp, _ := get(v, "calculated_price").(*dal.Record); cp == nil {
			continue
		}
		product, _ := get(v, "product").(*dal.Record)
		productID := get(v, "product_id")
		if productID == nil || productID == "" {
			productID = get(product, "id")
		}
		if productID == nil || productID == "" {
			continue
		}
		items = append(items, products.TaxedVariant{Variant: v, ProductID: productID, ProductTypeID: get(product, "type_id")})
	}
	if len(items) == 0 {
		return nil
	}
	return products.ApplyVariantTaxes(req, lineContext, items)
}

// Graph runs the variant query.graph of the routes.
func Graph(req *medusahttp.MedusaRequest, filters *dal.Record, pagination *dal.Record) (*query.GraphResult, bool, error) {
	fields, withInventory := productmw.PrepareInventoryQuantityFields(req.QueryConfig.Fields, "")
	req.QueryConfig.Fields = fields
	r, err := httputils.QueryOf(req.Scope).Graph(req.Ctx(), query.GraphInput{Entity: "variant", Fields: req.QueryConfig.Fields, Filters: filters, Pagination: pagination, Context: PricingContext(req)}, joiner.Options{})
	return r, withInventory, err
}

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	r, withInventory, err := Graph(req, req.FilterableFields, req.QueryConfig.Pagination)
	if err != nil {
		return err
	}
	variants := r.Data
	if variants == nil {
		variants = []any{}
	}
	if withInventory {
		if err := products.WrapInventoryQuantity(req, variants); err != nil {
			return err
		}
	}
	if err := WrapVariantsWithTaxPrices(req, variants); err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("variants", variants)
	for _, p := range [][2]string{{"count", "count"}, {"offset", "skip"}, {"limit", "take"}} {
		v := get(r.Metadata, p[1])
		if v == nil {
			v = float64(0)
		}
		out.Set(p[0], v)
	}
	return res.JSON(out)
}

// Route is route.ts.
var Route = medusahttp.Route{GET: GET}
