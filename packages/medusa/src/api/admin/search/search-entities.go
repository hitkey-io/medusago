// Package search mirrors api/admin/search.
package search

import (
	"regexp"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// AdminSearchEntityConfig mirrors AdminSearchEntityConfig.
type AdminSearchEntityConfig struct {
	Name string
	// GraphEntity is the query.graph entry point (Name when empty).
	GraphEntity string
	Fields      []string
	// Filters are always applied.
	Filters func() *dal.Record
	// TransformQuery normalizes `q` before it is passed as a filter.
	TransformQuery func(q string) string
}

func filters(kv ...any) func() *dal.Record {
	return func() *dal.Record {
		r := dal.NewRecord()
		for i := 0; i+1 < len(kv); i += 2 {
			r.Set(kv[i].(string), kv[i+1])
		}
		return r
	}
}

var leadingHash = regexp.MustCompile(`^#`)

// AdminSearchEntities mirrors ADMIN_SEARCH_ENTITIES.
var AdminSearchEntities = []AdminSearchEntityConfig{
	{
		Name:    "order",
		Fields:  []string{"id", "display_id", "email"},
		Filters: filters("is_draft_order", false),
		// Display shows `#${display_id}`; users often type the leading `#`.
		TransformQuery: func(q string) string { return leadingHash.ReplaceAllString(q, "") },
	},
	{Name: "product", Fields: []string{"id", "title", "thumbnail"}},
	{Name: "product_variant", Fields: []string{"id", "title", "sku", "product_id"}},
	{Name: "product_category", Fields: []string{"id", "name"}},
	{Name: "product_collection", Fields: []string{"id", "title"}},
	{Name: "customer", Fields: []string{"id", "email", "first_name", "last_name"}},
	{Name: "customer_group", Fields: []string{"id", "name"}},
	{Name: "inventory_item", Fields: []string{"id", "title", "sku"}},
	{Name: "promotion", Fields: []string{"id", "code", "status"}},
	{Name: "campaign", Fields: []string{"id", "name"}},
	{Name: "price_list", Fields: []string{"id", "title"}},
	{Name: "user", Fields: []string{"id", "email", "first_name", "last_name"}},
	{Name: "region", Fields: []string{"id", "name"}},
	{Name: "tax_region", Fields: []string{"id", "country_code", "province_code"}},
	{Name: "return_reason", Fields: []string{"id", "label", "value"}},
	{Name: "sales_channel", Fields: []string{"id", "name"}},
	{Name: "product_type", Fields: []string{"id", "value"}},
	{Name: "product_tag", Fields: []string{"id", "value"}},
	{Name: "stock_location", Fields: []string{"id", "name"}},
	{Name: "shipping_profile", Fields: []string{"id", "name"}},
	{
		// graphEntity is shared with secret_api_key below; disambiguated by
		// name since ADMIN_SEARCH_ENTITY_MAP and `entity=` both key on it.
		Name:        "publishable_api_key",
		GraphEntity: "api_key",
		Fields:      []string{"id", "title", "redacted"},
		Filters:     filters("type", "publishable"),
	},
	{
		Name:        "secret_api_key",
		GraphEntity: "api_key",
		Fields:      []string{"id", "title", "redacted"},
		Filters:     filters("type", "secret"),
	},
}

// AdminSearchEntityMap mirrors ADMIN_SEARCH_ENTITY_MAP.
var AdminSearchEntityMap = func() map[string]*AdminSearchEntityConfig {
	out := map[string]*AdminSearchEntityConfig{}
	for i := range AdminSearchEntities {
		out[AdminSearchEntities[i].Name] = &AdminSearchEntities[i]
	}
	return out
}()

// DefaultAdminSearchEntityNames mirrors DEFAULT_ADMIN_SEARCH_ENTITY_NAMES.
var DefaultAdminSearchEntityNames = func() []string {
	out := make([]string, len(AdminSearchEntities))
	for i, e := range AdminSearchEntities {
		out[i] = e.Name
	}
	return out
}()
