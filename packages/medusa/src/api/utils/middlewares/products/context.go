package products

import (
	"fmt"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"strings"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/caching"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// DefaultPriceFieldPaths mirrors DEFAULT_PRICE_FIELD_PATHS.
var DefaultPriceFieldPaths = []string{"variants.calculated_price"}

func get(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}

func filterable(req *medusahttp.MedusaRequest) *dal.Record {
	if req.FilterableFields == nil {
		req.FilterableFields = dal.NewRecord()
	}
	return req.FilterableFields
}

func toStrings(v any) []any {
	switch t := v.(type) {
	case nil:
		return nil
	case []any:
		return t
	case []string:
		out := make([]any, len(t))
		for i, s := range t {
			out[i] = s
		}
		return out
	}
	return []any{v}
}

// TransformAndValidateSalesChannelIDs mirrors
// transformAndValidateSalesChannelIds.
func TransformAndValidateSalesChannelIDs(req *medusahttp.MedusaRequest) ([]any, error) {
	fromKey := toStrings(get(req.PublishableKeyContext, "sales_channel_ids"))
	fromReq := toStrings(get(req.ValidatedQuery, "sales_channel_id"))
	if len(fromReq) > 0 {
		allowed := map[any]bool{}
		for _, id := range fromKey {
			allowed[id] = true
		}
		for _, id := range fromReq {
			if !allowed[id] {
				return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Requested sales channel is not part of the publishable key")
			}
		}
		return fromReq, nil
	}
	return fromKey, nil
}

// FilterByValidSalesChannels mirrors filterByValidSalesChannels.
func FilterByValidSalesChannels() medusahttp.MiddlewareFunction {
	return func(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, next medusahttp.NextFunction) error {
		ids, err := TransformAndValidateSalesChannelIDs(req)
		if err != nil {
			return err
		}
		if len(ids) == 0 {
			return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Publishable key needs to have a sales channel configured")
		}
		filterable(req).Set("sales_channel_id", ids)
		next(nil)
		return nil
	}
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func withCalculatedPrice(fields []string, paths []string) bool {
	for _, f := range fields {
		for _, p := range paths {
			if f == p || strings.HasPrefix(f, p+".") {
				return true
			}
		}
	}
	return false
}

// NormalizeDataForContext mirrors normalizeDataForContext.
func NormalizeDataForContext(paths ...string) medusahttp.MiddlewareFunction {
	if len(paths) == 0 {
		paths = DefaultPriceFieldPaths
	}
	return func(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, next medusahttp.NextFunction) error {
		with := false
		fields := make([]string, len(req.QueryConfig.Fields))
		for i, field := range req.QueryConfig.Fields {
			fields[i] = field
			for _, p := range paths {
				if field == p {
					with = true
					fields[i] = p + ".*"
					break
				}
				if strings.HasPrefix(field, p+".") {
					with = true
					break
				}
			}
		}
		req.QueryConfig.Fields = fields
		ff := filterable(req)
		if !with && filterIsPresent(get(ff, "region_id")) {
			for _, p := range paths {
				if !contains(req.QueryConfig.Fields, p+".*") {
					req.QueryConfig.Fields = append(req.QueryConfig.Fields, p+".*")
				}
			}
			with = len(paths) > 0
		}
		if !with {
			next(nil)
			return nil
		}
		regionID := get(ff, "region_id")
		countryCode := get(ff, "country_code")
		province := get(ff, "province")
		if cartID := get(ff, "cart_id"); filterIsPresent(cartID) {
			cart, err := httputils.RefetchEntity(req.Ctx(), req.Scope, "cart", cartID, []string{"region_id", "shipping_address.*"})
			if err != nil {
				return err
			}
			if c, ok := cart.(*dal.Record); ok {
				if r := get(c, "region_id"); filterIsPresent(r) {
					regionID = r
				}
				if addr, ok := get(c, "shipping_address").(*dal.Record); ok {
					countryCode = get(addr, "country_code")
					province = get(addr, "province")
				}
			}
		}
		if !filterIsPresent(regionID) {
			stores, err := httputils.RefetchEntities(req.Ctx(), req.Scope, httputils.RefetchEntitiesInput{Entity: "store", Fields: []string{"id", "default_region_id"}, Options: joiner.Options{Cache: caching.Enabled()}})
			if err != nil {
				return err
			}
			if len(stores.Data) > 0 {
				regionID = get(stores.Data[0].(*dal.Record), "default_region_id")
			}
		}
		if !filterIsPresent(regionID) {
			return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Missing required pricing context to calculate prices - region_id")
		}
		ff.Set("region_id", regionID)
		ff.Set("country_code", countryCode)
		ff.Set("province", province)
		next(nil)
		return nil
	}
}

func filterIsPresent(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case string:
		return t != ""
	case []any:
		return len(t) > 0
	case *dal.Record:
		return t.Len() > 0
	}
	return true
}

// SetPricingContext mirrors setPricingContext.
func SetPricingContext(paths ...string) medusahttp.MiddlewareFunction {
	if len(paths) == 0 {
		paths = DefaultPriceFieldPaths
	}
	return func(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, next medusahttp.NextFunction) error {
		if !withCalculatedPrice(req.QueryConfig.Fields, paths) {
			next(nil)
			return nil
		}
		regionID := get(filterable(req), "region_id")
		region, err := httputils.RefetchEntityWithOptions(req.Ctx(), req.Scope, "region", regionID, []string{"id", "currency_code"}, joiner.Options{Cache: caching.Enabled()})
		if err != nil {
			return err
		}
		reg, _ := region.(*dal.Record)
		if reg == nil {
			return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Region with id %v not found when populating the pricing context", jsString(regionID)))
		}
		pc := dal.NewRecord()
		pc.Set("region_id", get(reg, "id"))
		pc.Set("currency_code", get(reg, "currency_code"))
		if actor := get(req.AuthContext, "actor_id"); filterIsPresent(actor) {
			cf := dal.NewRecord()
			cust := dal.NewRecord()
			cust.Set("id", actor)
			cf.Set("customers", cust)
			groups, err := httputils.RefetchEntities(req.Ctx(), req.Scope, httputils.RefetchEntitiesInput{Entity: "customer_group", IDOrFilter: cf, Fields: []string{"id"}})
			if err != nil {
				return err
			}
			gs := []any{}
			for _, g := range groups.Data {
				x := dal.NewRecord()
				x.Set("id", get(g.(*dal.Record), "id"))
				gs = append(gs, x)
			}
			c := dal.NewRecord()
			c.Set("groups", gs)
			pc.Set("customer", c)
		}
		req.PricingContext = pc
		next(nil)
		return nil
	}
}

func jsString(v any) string {
	if v == nil {
		return "undefined"
	}
	return fmt.Sprint(v)
}

// SetTaxContext mirrors setTaxContext.
func SetTaxContext(paths ...string) medusahttp.MiddlewareFunction {
	if len(paths) == 0 {
		paths = DefaultPriceFieldPaths
	}
	return func(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, next medusahttp.NextFunction) error {
		if !withCalculatedPrice(req.QueryConfig.Fields, paths) {
			next(nil)
			return nil
		}
		ff := filterable(req)
		regionID := get(ff, "region_id")
		region, err := httputils.RefetchEntity(req.Ctx(), req.Scope, "region", regionID, []string{"automatic_taxes"})
		if err != nil {
			return err
		}
		reg, _ := region.(*dal.Record)
		if reg == nil {
			return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Region with id %v not found when populating the tax context", jsString(regionID)))
		}
		automatic, _ := get(reg, "automatic_taxes").(bool)
		if !automatic {
			next(nil)
			return nil
		}
		inclusivity := dal.NewRecord()
		inclusivity.Set("automaticTaxes", automatic)
		tc := dal.NewRecord()
		if filterIsPresent(get(ff, "country_code")) {
			addr := dal.NewRecord()
			addr.Set("country_code", get(ff, "country_code"))
			addr.Set("province_code", get(ff, "province"))
			lc := dal.NewRecord()
			lc.Set("address", addr)
			if req.Locale != "" {
				lc.Set("locale", req.Locale)
			}
			tc.Set("taxLineContext", lc)
		} else {
			tc.Set("taxLineContext", nil)
		}
		tc.Set("taxInclusivityContext", inclusivity)
		req.TaxContext = tc
		next(nil)
		return nil
	}
}
