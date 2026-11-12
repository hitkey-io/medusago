package utils

import (
	"sync"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// ComputedColumnDefinition mirrors ComputedColumnDefinition: a column
// injected for an entity in addition to the columns discovered from its
// schema. A computed display column has a RenderMode and RequiredFields, a
// filter-only column a "filter" Context and a Filter. Empty strings and nil
// values stand for undefined.
type ComputedColumnDefinition struct {
	// ID is the unique identifier of the column (e.g. "customer_display").
	ID string
	// Name is the display name of the column.
	Name string
	// RenderMode is required for display columns.
	RenderMode string
	// RequiredFields are the fields required to render the column.
	RequiredFields []string
	// OptionalFields are optional fields that enhance the rendering.
	OptionalFields []string
	// DefaultVisible is whether the column is visible by default.
	DefaultVisible bool
	// Description of the column.
	Description string
	// Category groups the columns (relationship, metadata, computed, ...).
	Category string
	// Metadata of the column.
	Metadata *dal.Record
	// Context is "display" (default), "filter" or "both".
	Context string
	// Filter is the filter configuration ({ enabled, operators?,
	// relationship?, enumValues? }).
	Filter *dal.Record
	// DataType is the column data type (default "string").
	DataType string
	// Sortable is whether the column is sortable (default false).
	Sortable bool
}

// rec builds an ordered record from key/value pairs.
func rec(kv ...any) *dal.Record {
	r := dal.NewRecord()
	for i := 0; i+1 < len(kv); i += 2 {
		r.Set(kv[i].(string), kv[i+1])
	}
	return r
}

// builtinComputedColumns mirrors BUILTIN_COMPUTED_COLUMNS (entities in
// declaration order).
func builtinComputedColumns() []struct {
	entity  string
	columns []*ComputedColumnDefinition
} {
	stockLocationFilter := func(filterKey string) *dal.Record {
		return rec(
			"enabled", true,
			"operators", []string{"in"},
			"relationship", rec(
				"entity", "StockLocation",
				"value_field", "id",
				"display_field", "name",
				"multiple", true,
				"endpoint", "/admin/stock-locations",
				"filter_key", filterKey,
			),
		)
	}
	return []struct {
		entity  string
		columns []*ComputedColumnDefinition
	}{
		{"Order", []*ComputedColumnDefinition{
			{
				ID:             "customer_display",
				Name:           "Customer",
				RenderMode:     "name",
				RequiredFields: []string{"customer.first_name", "customer.last_name", "customer.email"},
				OptionalFields: []string{"customer.phone"},
				DefaultVisible: true,
				Description:    "Customer name and contact information",
				Category:       "relationship",
				Metadata: rec(
					"name_source", "customer",
					"fallback_fields", []string{"customer.email", "customer.phone"},
					"empty_label_key", "customers.guest",
					"empty_label", "Guest",
				),
			},
			{
				ID:             "shipping_address_display",
				Name:           "Shipping Address",
				RenderMode:     "address",
				RequiredFields: []string{"shipping_address.city", "shipping_address.country_code"},
				OptionalFields: []string{"shipping_address.address_1", "shipping_address.province", "shipping_address.postal_code"},
				Metadata:       rec("address_field", "shipping_address"),
				DefaultVisible: false,
				Description:    "Shipping address summary",
				Category:       "relationship",
			},
			{
				ID:             "billing_address_display",
				Name:           "Billing Address",
				RenderMode:     "address",
				RequiredFields: []string{"billing_address.city", "billing_address.country_code"},
				OptionalFields: []string{"billing_address.address_1", "billing_address.province", "billing_address.postal_code"},
				Metadata:       rec("address_field", "billing_address"),
				DefaultVisible: false,
				Description:    "Billing address summary",
				Category:       "relationship",
			},
			{
				ID:             "order_shipping_country_display",
				Name:           "Country",
				RenderMode:     "country_code",
				RequiredFields: []string{"shipping_address.country_code"},
				OptionalFields: []string{},
				DefaultVisible: true,
				Description:    "Shipping country",
				Category:       "metadata",
				Metadata:       rec("country_code_field", "shipping_address.country_code"),
			},
		}},
		{"Product", []*ComputedColumnDefinition{
			{
				ID:             "product_display",
				Name:           "Product",
				RenderMode:     "product_info",
				RequiredFields: []string{"title", "thumbnail"},
				OptionalFields: []string{"handle"},
				DefaultVisible: true,
				Description:    "Product title and thumbnail",
				Category:       "computed",
			},
			{
				ID:             "variants_count",
				Name:           "Variants",
				RenderMode:     "count",
				RequiredFields: []string{"variants.id"},
				OptionalFields: []string{},
				DefaultVisible: true,
				Description:    "Number of product variants",
				Category:       "metric",
				Metadata:       rec("list_field", "variants"),
			},
			{
				ID:             "categories_display",
				Name:           "Categories",
				RenderMode:     "badges",
				RequiredFields: []string{"categories.name"},
				OptionalFields: []string{},
				DefaultVisible: false,
				Description:    "Product categories",
				Category:       "relationship",
				Metadata:       rec("display_field", "name", "list_field", "categories"),
			},
			{
				ID:             "sales_channels_display",
				Name:           "Sales Channels",
				RenderMode:     "badges",
				RequiredFields: []string{"sales_channels.name"},
				OptionalFields: []string{},
				DefaultVisible: true,
				Description:    "Sales channels the product is available in",
				Category:       "relationship",
				Metadata:       rec("list_field", "sales_channels", "display_field", "name"),
			},
		}},
		{"Customer", []*ComputedColumnDefinition{
			{
				ID:             "customer_name",
				Name:           "Name",
				RenderMode:     "name",
				RequiredFields: []string{"first_name", "last_name"},
				OptionalFields: []string{},
				DefaultVisible: true,
				Description:    "Customer's full name",
				Category:       "computed",
			},
		}},
		{"CustomerGroup", []*ComputedColumnDefinition{
			{
				ID:             "customers_count",
				Name:           "Customers",
				RenderMode:     "count",
				RequiredFields: []string{"customers.id"},
				OptionalFields: []string{},
				DefaultVisible: true,
				Description:    "Number of customers in the group",
				Category:       "metric",
				Metadata:       rec("list_field", "customers"),
			},
		}},
		{"PriceList", []*ComputedColumnDefinition{
			{
				ID:             "price_overrides",
				Name:           "Price overrides",
				RenderMode:     "price_overrides_count",
				RequiredFields: []string{},
				OptionalFields: []string{},
				DefaultVisible: true,
				Description:    "Number of price overrides in the price list",
				Category:       "metric",
			},
		}},
		{"ProductCollection", []*ComputedColumnDefinition{
			{
				ID:             "products_count",
				Name:           "Products",
				RenderMode:     "count",
				RequiredFields: []string{"products.id"},
				OptionalFields: []string{},
				DefaultVisible: true,
				Description:    "Number of products in the collection",
				Category:       "metric",
				Metadata:       rec("list_field", "products"),
			},
		}},
		{"ProductOption", []*ComputedColumnDefinition{
			{
				ID:             "values_count",
				Name:           "Values",
				RenderMode:     "count",
				RequiredFields: []string{"values.id"},
				OptionalFields: []string{},
				DefaultVisible: true,
				Description:    "Number of values for the option",
				Category:       "metric",
				Metadata:       rec("list_field", "values"),
			},
		}},
		{"Promotion", []*ComputedColumnDefinition{
			{
				ID:             "method",
				Name:           "Method",
				RenderMode:     "promotion_method",
				RequiredFields: []string{"is_automatic"},
				OptionalFields: []string{},
				DefaultVisible: true,
				Description:    "Whether the promotion is automatic or code based",
				Category:       "computed",
			},
			{
				ID:         "status_display",
				Name:       "Status",
				RenderMode: "status",
				RequiredFields: []string{
					"status",
					"campaign.starts_at",
					"campaign.ends_at",
					"campaign.budget.type",
					"campaign.budget.limit",
					"campaign.budget.used",
				},
				OptionalFields: []string{},
				DefaultVisible: true,
				Description:    "Promotion status (derived from status and campaign)",
				Metadata:       rec("resolver", "promotion_status"),
				Category:       "computed",
			},
		}},
		{"Region", []*ComputedColumnDefinition{
			{
				ID:             "region_country_display",
				Name:           "Countries",
				RenderMode:     "region_countries",
				RequiredFields: []string{"countries.iso_2"},
				OptionalFields: []string{},
				DefaultVisible: true,
				Description:    "Countries in the region",
				Category:       "relationship",
			},
			{
				ID:             "payment_providers_display",
				Name:           "Payment providers",
				RenderMode:     "region_payment_providers",
				RequiredFields: []string{"payment_providers.id"},
				OptionalFields: []string{},
				DefaultVisible: true,
				Description:    "Payment providers enabled for the region",
				Category:       "relationship",
			},
		}},
		{"StockLocation", []*ComputedColumnDefinition{
			{
				ID:             "location_address_display",
				Name:           "Address",
				RenderMode:     "address",
				RequiredFields: []string{"address.*"},
				OptionalFields: []string{},
				Metadata:       rec("address_field", "address"),
				DefaultVisible: true,
				Description:    "Formatted stock location address",
				Category:       "relationship",
			},
			{
				ID:             "location_country_display",
				Name:           "Country",
				RenderMode:     "country_code",
				RequiredFields: []string{"address.country_code"},
				OptionalFields: []string{},
				DefaultVisible: true,
				Description:    "Stock location country",
				Category:       "metadata",
				Metadata:       rec("country_code_field", "address.country_code"),
			},
			{
				ID:             "shipping_fulfillment",
				Name:           "Shipping",
				RenderMode:     "stock_location_shipping",
				RequiredFields: []string{"fulfillment_sets.type"},
				OptionalFields: []string{},
				DefaultVisible: true,
				Description:    "Whether shipping fulfillment is enabled",
				Category:       "metadata",
			},
			{
				ID:             "pickup_fulfillment",
				Name:           "Pickup",
				RenderMode:     "stock_location_pickup",
				RequiredFields: []string{"fulfillment_sets.type"},
				OptionalFields: []string{},
				DefaultVisible: true,
				Description:    "Whether pickup fulfillment is enabled",
				Category:       "metadata",
			},
			{
				ID:             "location_sales_channels",
				Name:           "Connected sales channels",
				RenderMode:     "stock_location_sales_channels",
				RequiredFields: []string{"sales_channels.name"},
				OptionalFields: []string{},
				DefaultVisible: true,
				Description:    "Sales channels served by the location",
				Category:       "relationship",
			},
		}},
		{"InventoryItem", []*ComputedColumnDefinition{
			{
				// Filter-only injected column: inventory items reach a location
				// through the `location_levels` array relationship, which the
				// generator can't turn into a usable "location" filter.
				ID:             "inventory_location_filter",
				Name:           "Location",
				Context:        "filter",
				DefaultVisible: false,
				Description:    "Stock location",
				Category:       "relationship",
				Filter:         stockLocationFilter("location_levels.location_id"),
			},
		}},
		{"ReservationItem", []*ComputedColumnDefinition{
			{
				ID:             "reservation_location_filter",
				Name:           "Location",
				Context:        "filter",
				DefaultVisible: false,
				Description:    "Stock location",
				Category:       "relationship",
				Filter:         stockLocationFilter("location_id"),
			},
		}},
	}
}

// ComputedColumnRegistry mirrors ComputedColumnRegistry: the computed
// columns per entity (built-in ones registered first).
type ComputedColumnRegistry struct {
	mu      sync.Mutex
	columns map[string][]*ComputedColumnDefinition
}

// NewComputedColumnRegistry mirrors new ComputedColumnRegistry().
func NewComputedColumnRegistry() *ComputedColumnRegistry {
	r := &ComputedColumnRegistry{columns: map[string][]*ComputedColumnDefinition{}}
	for _, e := range builtinComputedColumns() {
		r.Register(e.entity, e.columns)
	}
	return r
}

// Register mirrors register: the columns are appended to the ones already
// registered for the entity; a column sharing an existing id replaces it
// (keeping its position).
func (r *ComputedColumnRegistry) Register(entity string, columns []*ComputedColumnDefinition) {
	r.mu.Lock()
	defer r.mu.Unlock()
	existing := r.columns[entity]
	out := append([]*ComputedColumnDefinition(nil), existing...)
	index := map[string]int{}
	for i, c := range out {
		index[c.ID] = i
	}
	for _, c := range columns {
		if i, ok := index[c.ID]; ok {
			out[i] = c
			continue
		}
		index[c.ID] = len(out)
		out = append(out, c)
	}
	r.columns[entity] = out
}

// Get mirrors get.
func (r *ComputedColumnRegistry) Get(entityName string) []*ComputedColumnDefinition {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]*ComputedColumnDefinition(nil), r.columns[entityName]...)
}

var (
	computedRegistryMu       sync.Mutex
	computedRegistryInstance *ComputedColumnRegistry
)

// GetComputedColumnRegistry mirrors getComputedColumnRegistry (singleton).
func GetComputedColumnRegistry() *ComputedColumnRegistry {
	computedRegistryMu.Lock()
	defer computedRegistryMu.Unlock()
	if computedRegistryInstance == nil {
		computedRegistryInstance = NewComputedColumnRegistry()
	}
	return computedRegistryInstance
}
