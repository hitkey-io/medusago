package services

import (
	"context"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

// ListBySuffix dispatches list<suffix> (listShippingOptions reroutes to
// listShippingOptionsForContext with a context or an address).
func (s *FulfillmentModuleService) ListBySuffix(ctx context.Context, suffix string, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]*dal.Record, error) {
	if suffix == "ShippingOptions" && filters != nil && (truthy(get(filters, "context")) || truthy(get(filters, "address"))) {
		return s.ListShippingOptionsForContext(ctx, filters, config)
	}
	return s.MedusaService.ListBySuffix(ctx, suffix, filters, config)
}

// ListAndCountBySuffix dispatches listAndCount<suffix>.
func (s *FulfillmentModuleService) ListAndCountBySuffix(ctx context.Context, suffix string, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]*dal.Record, int, error) {
	return s.MedusaService.ListAndCountBySuffix(ctx, suffix, filters, config)
}

func jsTemplate(v any, ok bool) string {
	if !ok {
		return "undefined"
	}
	switch t := v.(type) {
	case nil:
		return "null"
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	case float64:
		if t == math.Trunc(t) && math.Abs(t) < 1e21 {
			return strconv.FormatFloat(t, 'f', -1, 64)
		}
		return strconv.FormatFloat(t, 'g', -1, 64)
	case []any:
		parts := make([]string, len(t))
		for i, x := range t {
			if x != nil {
				parts[i] = jsTemplate(x, true)
			}
		}
		return strings.Join(parts, ",")
	case *dal.Record:
		return "[object Object]"
	}
	return ""
}

func jsNumber(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return math.NaN()
	}
	return f
}

func parseDate(s string) (time.Time, bool) {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05.000Z", "2006-01-02", time.RFC1123} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func isDate(s string) bool {
	if !math.IsNaN(jsNumber(s)) {
		return false
	}
	_, ok := parseDate(s)
	return ok
}

func compare(ctxValue, ruleValue string, cmp func(a, b float64) bool) bool {
	if isDate(ctxValue) && isDate(ruleValue) {
		a, _ := parseDate(ctxValue)
		b, _ := parseDate(ruleValue)
		return cmp(float64(a.UnixMilli()), float64(b.UnixMilli()))
	}
	return cmp(jsNumber(ctxValue), jsNumber(ruleValue))
}

// isContextValid mirrors isContextValid (every rule).
func isContextValid(c *dal.Record, rules []*dal.Record) bool {
	for _, rule := range rules {
		v, ok := dal.PickValueFromObject(fmtString(get(rule, "attribute")), c)
		ctxValue := jsTemplate(v, ok)
		value := get(rule, "value")
		var valid bool
		switch fmtString(get(rule, "operator")) {
		case "in", "nin":
			found := false
			if list, isList := value.([]any); isList {
				for _, x := range list {
					if x == ctxValue {
						found = true
					}
				}
			}
			valid = found == (fmtString(get(rule, "operator")) == "in")
		case "eq":
			valid = value == ctxValue
		case "ne":
			valid = value != ctxValue
		case "gt":
			valid = compare(ctxValue, fmtString(value), func(a, b float64) bool { return a > b })
		case "gte":
			valid = compare(ctxValue, fmtString(value), func(a, b float64) bool { return a >= b })
		case "lt":
			valid = compare(ctxValue, fmtString(value), func(a, b float64) bool { return a < b })
		case "lte":
			valid = compare(ctxValue, fmtString(value), func(a, b float64) bool { return a <= b })
		}
		if !valid {
			return false
		}
	}
	return true
}

func fmtString(v any) string {
	s, _ := v.(string)
	return s
}

func isPresentValue(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case string:
		return t != ""
	}
	return true
}

// buildGeoZoneConstraintsFromAddress mirrors buildGeoZoneConstraintsFromAddress.
func buildGeoZoneConstraintsFromAddress(address *dal.Record) []any {
	hierarchy := []struct {
		prop  string
		props []string
		typ   string
	}{
		{"postal_expression", []string{"country_code", "province_code", "city", "postal_expression"}, "zip"},
		{"city", []string{"country_code", "province_code", "city"}, "city"},
		{"province_code", []string{"country_code", "province_code"}, "province"},
		{"country_code", []string{"country_code"}, "country"},
	}
	var out []any
	for _, h := range hierarchy {
		if !truthy(get(address, h.prop)) {
			continue
		}
		c := dal.NewRecord()
		c.Set("type", h.typ)
		for _, p := range h.props {
			if v := get(address, p); isPresentValue(v) {
				c.Set(p, v)
			}
		}
		out = append(out, c)
	}
	return out
}

// ListShippingOptionsForContext mirrors listShippingOptionsForContext.
func (s *FulfillmentModuleService) ListShippingOptionsForContext(ctx context.Context, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]*dal.Record, error) {
	where := dal.NewRecord()
	for _, k := range filters.Keys() {
		switch k {
		case "fulfillment_set_id", "fulfillment_set_type", "address", "context":
			continue
		}
		v, _ := filters.Get(k)
		where.Set(k, v)
	}
	ctxRec, _ := get(filters, "context").(*dal.Record)
	relations := append([]string{"rules", "type", "shipping_profile", "provider"}, config.Relations...)
	if ctxRec != nil && config.Take == nil {
		// take: null
		config.Take = nil
	}
	setID, setType := get(filters, "fulfillment_set_id"), get(filters, "fulfillment_set_type")
	if truthy(setID) || truthy(setType) {
		sz, _ := get(where, "service_zone").(*dal.Record)
		if sz == nil {
			sz = dal.NewRecord()
		} else {
			sz = clone(sz)
		}
		fs, _ := get(sz, "fulfillment_set").(*dal.Record)
		if fs == nil {
			fs = dal.NewRecord()
		} else {
			fs = clone(fs)
		}
		if truthy(setID) {
			fs.Set("id", setID)
		}
		if truthy(setType) {
			fs.Set("type", setType)
		}
		sz.Set("fulfillment_set", fs)
		where.Set("service_zone", sz)
		relations = append(relations, "service_zone.fulfillment_set")
	}
	if address, ok := get(filters, "address").(*dal.Record); ok && address != nil {
		constraints := buildGeoZoneConstraintsFromAddress(address)
		if len(constraints) > 0 {
			sz, _ := get(where, "service_zone").(*dal.Record)
			if sz == nil {
				sz = dal.NewRecord()
			} else {
				sz = clone(sz)
			}
			existing, _ := get(sz, "geo_zones").(*dal.Record)
			var ors []any
			for _, c := range constraints {
				m := dal.NewRecord()
				if existing != nil {
					for _, k := range existing.Keys() {
						v, _ := existing.Get(k)
						m.Set(k, v)
					}
				}
				cr := c.(*dal.Record)
				for _, k := range cr.Keys() {
					v, _ := cr.Get(k)
					m.Set(k, v)
				}
				ors = append(ors, m)
			}
			gz := dal.NewRecord()
			gz.Set("$or", ors)
			sz.Set("geo_zones", gz)
			where.Set("service_zone", sz)
			relations = append(relations, "service_zone.geo_zones")
		}
	}
	seen := map[string]bool{}
	var rels []string
	for _, r := range relations {
		if !seen[r] {
			seen[r] = true
			rels = append(rels, r)
		}
	}
	config.Relations = rels
	options, err := s.Internal("ShippingOption").List(ctx, where, config)
	if err != nil {
		return nil, err
	}
	if ctxRec == nil {
		return options, nil
	}
	var out []*dal.Record
	for _, o := range options {
		rules := records(get(o, "rules"))
		if len(rules) == 0 || isContextValid(ctxRec, rules) {
			out = append(out, o)
		}
	}
	if out == nil {
		out = []*dal.Record{}
	}
	return out, nil
}
