package utils

import (
	"sync"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// EntityOverride mirrors EntityOverride: how the columns of an entity are
// generated and displayed. Nil slices and maps stand for undefined.
type EntityOverride struct {
	// ExcludeFields are specific field names to exclude.
	ExcludeFields []string
	// ExcludeSuffixes are field name suffixes to exclude.
	ExcludeSuffixes []string
	// ExcludePrefixes are field name prefixes to exclude.
	ExcludePrefixes []string
	// DefaultVisibleFields are the fields shown by default (in order).
	DefaultVisibleFields []string
	// DefaultFieldOrdering is the ordering of fields (lower first).
	DefaultFieldOrdering map[string]float64
	// FieldRenderModes overrides the inferred render mode of field paths.
	FieldRenderModes map[string]string
	// FieldMetadata is the per-column renderer configuration.
	FieldMetadata map[string]*dal.Record
	// AdditionalTypes are GraphQL types whose fields are included too.
	AdditionalTypes []string
	// NonFilterableFields are displayed but cannot be filtered on.
	NonFilterableFields []string
	// NonSortableFields are displayed but cannot be sorted on.
	NonSortableFields []string
	// ComputedColumns are computed columns specific to the entity.
	ComputedColumns []*ComputedColumnDefinition
}

// std returns an override with the exclusions every built-in override has
// (excludeSuffixes ["_link"], excludePrefixes ["raw_"], excludeFields []).
func std(o EntityOverride) EntityOverride {
	o.ExcludeSuffixes, o.ExcludePrefixes = []string{"_link"}, []string{"raw_"}
	if o.ExcludeFields == nil {
		o.ExcludeFields = []string{}
	}
	return o
}

// builtinEntityOverrides mirrors BUILTIN_ENTITY_OVERRIDES (entities in
// declaration order).
func builtinEntityOverrides() []struct {
	entity   string
	override EntityOverride
} {
	return []struct {
		entity   string
		override EntityOverride
	}{
		{"Order", std(EntityOverride{
			ExcludeFields:   []string{"order_change"},
			AdditionalTypes: []string{"OrderDetail"},
			DefaultVisibleFields: []string{
				"display_id", "created_at", "payment_status", "fulfillment_status", "total",
				"customer_display", "order_shipping_country_display", "sales_channel.name",
			},
			DefaultFieldOrdering: map[string]float64{
				"display_id":                     100,
				"custom_display_id":              101,
				"created_at":                     200,
				"customer_display":               300,
				"sales_channel.name":             400,
				"payment_status":                 500,
				"fulfillment_status":             600,
				"total":                          700,
				"order_shipping_country_display": 800,
			},
			NonFilterableFields: []string{"payment_status", "fulfillment_status"},
			FieldMetadata: map[string]*dal.Record{
				"payment_status":     rec("resolver", "order_payment_status"),
				"fulfillment_status": rec("resolver", "order_fulfillment_status"),
			},
		})},
		{"Product", std(EntityOverride{
			DefaultVisibleFields: []string{"product_display", "collection.title", "sales_channels_display", "variants_count", "status"},
			DefaultFieldOrdering: map[string]float64{
				"product_display":        100,
				"collection.title":       200,
				"sales_channels_display": 300,
				"variants_count":         400,
				"status":                 500,
			},
			FieldMetadata: map[string]*dal.Record{"status": rec("resolver", "product_status")},
		})},
		{"Customer", std(EntityOverride{
			DefaultVisibleFields: []string{"email", "customer_name", "has_account", "created_at"},
			DefaultFieldOrdering: map[string]float64{"email": 100, "customer_name": 200, "has_account": 300, "created_at": 400},
		})},
		{"CustomerGroup", std(EntityOverride{
			DefaultVisibleFields: []string{"name", "customers_count", "created_at", "updated_at"},
			DefaultFieldOrdering: map[string]float64{"name": 100, "customers_count": 200, "created_at": 300, "updated_at": 400},
		})},
		{"PriceList", std(EntityOverride{
			DefaultVisibleFields: []string{"title", "status", "price_overrides"},
			DefaultFieldOrdering: map[string]float64{"title": 100, "status": 200, "price_overrides": 300},
			FieldMetadata:        map[string]*dal.Record{"status": rec("resolver", "price_list_status")},
		})},
		{"ProductCollection", std(EntityOverride{
			DefaultVisibleFields: []string{"title", "handle", "products_count"},
			DefaultFieldOrdering: map[string]float64{"title": 100, "handle": 200, "products_count": 300},
			FieldRenderModes:     map[string]string{"handle": "handle"},
		})},
		{"ProductOption", std(EntityOverride{
			DefaultVisibleFields: []string{"title", "values_count", "is_exclusive"},
			DefaultFieldOrdering: map[string]float64{"title": 100, "values_count": 200, "is_exclusive": 300},
			FieldRenderModes:     map[string]string{"is_exclusive": "product_option_exclusivity"},
		})},
		{"ProductOptionValue", std(EntityOverride{
			DefaultVisibleFields: []string{"value"},
			DefaultFieldOrdering: map[string]float64{"value": 100},
		})},
		{"InventoryItem", std(EntityOverride{
			DefaultVisibleFields: []string{"title", "sku", "reserved_quantity", "stocked_quantity"},
			DefaultFieldOrdering: map[string]float64{"title": 100, "sku": 200, "reserved_quantity": 300, "stocked_quantity": 400},
			// Quantities are rendered with the item's unit of measure, which
			// lives on the row itself.
			FieldRenderModes: map[string]string{"reserved_quantity": "quantity", "stocked_quantity": "quantity"},
		})},
		{"User", std(EntityOverride{
			DefaultVisibleFields: []string{"email", "first_name", "last_name", "created_at", "updated_at"},
			DefaultFieldOrdering: map[string]float64{"email": 100, "first_name": 200, "last_name": 300, "created_at": 400, "updated_at": 500},
		})},
		{"Region", std(EntityOverride{
			DefaultVisibleFields: []string{"name", "region_country_display", "payment_providers_display"},
			DefaultFieldOrdering: map[string]float64{"name": 100, "region_country_display": 200, "payment_providers_display": 300},
		})},
		{"Promotion", std(EntityOverride{
			DefaultVisibleFields: []string{"code", "method", "status_display"},
			DefaultFieldOrdering: map[string]float64{"code": 100, "method": 200, "status_display": 300},
		})},
		{"Campaign", std(EntityOverride{
			DefaultVisibleFields: []string{"name", "description", "campaign_identifier", "starts_at", "ends_at"},
			DefaultFieldOrdering: map[string]float64{"name": 100, "description": 200, "campaign_identifier": 300, "starts_at": 400, "ends_at": 500},
		})},
		{"SalesChannel", std(EntityOverride{
			DefaultVisibleFields: []string{"name", "description", "is_disabled", "created_at", "updated_at"},
			DefaultFieldOrdering: map[string]float64{"name": 100, "description": 200, "is_disabled": 300, "created_at": 400, "updated_at": 500},
			// is_disabled would infer as boolean; force the status renderer and
			// route it through the sales_channel_status resolver.
			FieldRenderModes: map[string]string{"is_disabled": "status"},
			FieldMetadata:    map[string]*dal.Record{"is_disabled": rec("resolver", "sales_channel_status")},
		})},
		{"ReturnReason", std(EntityOverride{
			DefaultVisibleFields: []string{"label", "value", "description"},
			DefaultFieldOrdering: map[string]float64{"label": 100, "value": 200, "description": 300},
		})},
		{"RefundReason", std(EntityOverride{
			DefaultVisibleFields: []string{"label", "code", "description"},
			DefaultFieldOrdering: map[string]float64{"label": 100, "code": 200, "description": 300},
		})},
		{"ProductType", std(EntityOverride{
			DefaultVisibleFields: []string{"value", "created_at", "updated_at"},
			DefaultFieldOrdering: map[string]float64{"value": 100, "created_at": 200, "updated_at": 300},
		})},
		{"ProductTag", std(EntityOverride{
			DefaultVisibleFields: []string{"value", "created_at", "updated_at"},
			DefaultFieldOrdering: map[string]float64{"value": 100, "created_at": 200, "updated_at": 300},
		})},
		{"ShippingProfile", std(EntityOverride{
			DefaultVisibleFields: []string{"name", "type"},
			DefaultFieldOrdering: map[string]float64{"name": 100, "type": 200},
		})},
		{"ShippingOptionType", std(EntityOverride{
			DefaultVisibleFields: []string{"label", "code", "description", "created_at", "updated_at"},
			DefaultFieldOrdering: map[string]float64{"label": 100, "code": 200, "description": 300, "created_at": 400, "updated_at": 500},
		})},
		{"ReservationItem", std(EntityOverride{
			DefaultVisibleFields: []string{"inventory_item.sku", "description", "created_at", "quantity"},
			DefaultFieldOrdering: map[string]float64{"inventory_item.sku": 100, "description": 200, "created_at": 300, "quantity": 400},
			FieldRenderModes:     map[string]string{"quantity": "quantity"},
			// A reservation's unit of measure belongs to the inventory item it
			// reserves, so point the renderer at the related item.
			FieldMetadata: map[string]*dal.Record{
				"quantity": rec("unit_of_measure_path", "inventory_item.unit_of_measure"),
			},
		})},
		{"StockLocation", std(EntityOverride{
			DefaultVisibleFields: []string{"name", "location_address_display", "shipping_fulfillment", "pickup_fulfillment", "location_sales_channels"},
			DefaultFieldOrdering: map[string]float64{
				"name":                     100,
				"location_address_display": 200,
				"shipping_fulfillment":     300,
				"pickup_fulfillment":       400,
				"location_sales_channels":  500,
			},
		})},
		{"ApiKey", std(EntityOverride{
			DefaultVisibleFields: []string{"title", "redacted", "type", "revoked_at", "last_used_at", "created_at"},
			DefaultFieldOrdering: map[string]float64{
				"title":        100,
				"redacted":     200,
				"type":         300,
				"revoked_at":   400,
				"last_used_at": 500,
				"created_at":   600,
			},
			// Token badge, type label, and active/revoked status all use
			// dedicated renderers registered by the api key table.
			FieldRenderModes: map[string]string{"redacted": "api_key_token", "type": "api_key_type", "revoked_at": "status"},
			// revoked_at drives active/revoked via the api_key_status resolver.
			FieldMetadata: map[string]*dal.Record{"revoked_at": rec("resolver", "api_key_status")},
		})},
	}
}

// BuiltinEntityOverrideNames are the entities of BUILTIN_ENTITY_OVERRIDES.
func BuiltinEntityOverrideNames() []string {
	var out []string
	for _, e := range builtinEntityOverrides() {
		out = append(out, e.entity)
	}
	return out
}

// deduplicate mirrors deduplicate (a Set of the items, first occurrence
// kept).
func deduplicate(items []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, s := range items {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

func deduplicateColumns(items []*ComputedColumnDefinition) []*ComputedColumnDefinition {
	out := []*ComputedColumnDefinition{}
	seen := map[*ComputedColumnDefinition]bool{}
	for _, c := range items {
		if !seen[c] {
			seen[c] = true
			out = append(out, c)
		}
	}
	return out
}

func concat[T any](a, b []T) []T {
	return append(append([]T{}, a...), b...)
}

func mergeMaps[V any](a, b map[string]V) map[string]V {
	out := map[string]V{}
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}

func coalesce(a, b []string) []string {
	if a != nil {
		return a
	}
	return b
}

// EntityOverrideRegistry mirrors EntityOverrideRegistry: the overrides per
// entity (built-in ones registered first).
type EntityOverrideRegistry struct {
	mu        sync.Mutex
	overrides map[string]*EntityOverride
	names     []string
}

// NewEntityOverrideRegistry mirrors new EntityOverrideRegistry().
func NewEntityOverrideRegistry() *EntityOverrideRegistry {
	r := &EntityOverrideRegistry{overrides: map[string]*EntityOverride{}}
	for _, e := range builtinEntityOverrides() {
		r.Register(e.entity, e.override)
	}
	return r
}

// Register mirrors register: an override of an entity that already has one
// is merged into it (new values take precedence, lists concatenated and
// deduplicated, maps merged).
func (r *EntityOverrideRegistry) Register(entityName string, override EntityOverride) {
	r.mu.Lock()
	defer r.mu.Unlock()
	existing, ok := r.overrides[entityName]
	if !ok {
		o := override
		r.overrides[entityName] = &o
		r.names = append(r.names, entityName)
		return
	}
	r.overrides[entityName] = &EntityOverride{
		ExcludeFields:        deduplicate(concat(existing.ExcludeFields, override.ExcludeFields)),
		ExcludeSuffixes:      coalesce(override.ExcludeSuffixes, existing.ExcludeSuffixes),
		ExcludePrefixes:      coalesce(override.ExcludePrefixes, existing.ExcludePrefixes),
		DefaultVisibleFields: coalesce(override.DefaultVisibleFields, existing.DefaultVisibleFields),
		DefaultFieldOrdering: mergeMaps(existing.DefaultFieldOrdering, override.DefaultFieldOrdering),
		FieldRenderModes:     mergeMaps(existing.FieldRenderModes, override.FieldRenderModes),
		AdditionalTypes:      deduplicate(concat(existing.AdditionalTypes, override.AdditionalTypes)),
		NonFilterableFields:  deduplicate(concat(existing.NonFilterableFields, override.NonFilterableFields)),
		NonSortableFields:    deduplicate(concat(existing.NonSortableFields, override.NonSortableFields)),
		FieldMetadata:        mergeMaps(existing.FieldMetadata, override.FieldMetadata),
		ComputedColumns:      deduplicateColumns(concat(existing.ComputedColumns, override.ComputedColumns)),
	}
}

// Get mirrors get (nil when the entity has no override).
func (r *EntityOverrideRegistry) Get(entityName string) *EntityOverride {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.overrides[entityName]
}

// Has mirrors has.
func (r *EntityOverrideRegistry) Has(entityName string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.overrides[entityName]
	return ok
}

// GetEntityNames mirrors getEntityNames (registration order).
func (r *EntityOverrideRegistry) GetEntityNames() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.names...)
}

// GetAll mirrors getAll.
func (r *EntityOverrideRegistry) GetAll() map[string]*EntityOverride {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := map[string]*EntityOverride{}
	for k, v := range r.overrides {
		out[k] = v
	}
	return out
}

var (
	overrideRegistryMu       sync.Mutex
	overrideRegistryInstance *EntityOverrideRegistry
)

// GetEntityOverrideRegistry mirrors getEntityOverrideRegistry (singleton).
func GetEntityOverrideRegistry() *EntityOverrideRegistry {
	overrideRegistryMu.Lock()
	defer overrideRegistryMu.Unlock()
	if overrideRegistryInstance == nil {
		overrideRegistryInstance = NewEntityOverrideRegistry()
	}
	return overrideRegistryInstance
}

// ResetEntityOverrideRegistry mirrors resetEntityOverrideRegistry (tests).
func ResetEntityOverrideRegistry() {
	overrideRegistryMu.Lock()
	defer overrideRegistryMu.Unlock()
	overrideRegistryInstance = nil
}

// GetEntityOverride mirrors getEntityOverride.
func GetEntityOverride(entityName string) *EntityOverride {
	return GetEntityOverrideRegistry().Get(entityName)
}

// HasEntityOverride mirrors hasEntityOverride.
func HasEntityOverride(entityName string) bool {
	return GetEntityOverrideRegistry().Has(entityName)
}

func resolveOverride(entityName string, override *EntityOverride) *EntityOverride {
	if override != nil {
		return override
	}
	return GetEntityOverride(entityName)
}

// GetFieldFilterRules mirrors getFieldFilterRules (defaults for what the
// override leaves undefined).
func GetFieldFilterRules(entityName string, override *EntityOverride) FieldFilterRules {
	o := resolveOverride(entityName, override)
	if o == nil {
		o = &EntityOverride{}
	}
	rules := FieldFilterRules{ExcludeSuffixes: o.ExcludeSuffixes, ExcludePrefixes: o.ExcludePrefixes, ExcludeFields: o.ExcludeFields}
	if rules.ExcludeSuffixes == nil {
		rules.ExcludeSuffixes = []string{"_link"}
	}
	if rules.ExcludePrefixes == nil {
		rules.ExcludePrefixes = []string{"raw_"}
	}
	if rules.ExcludeFields == nil {
		rules.ExcludeFields = []string{}
	}
	return rules
}

// GetDefaultVisibleFields mirrors getDefaultVisibleFields.
func GetDefaultVisibleFields(entityName string, override *EntityOverride) []string {
	if o := resolveOverride(entityName, override); o != nil && o.DefaultVisibleFields != nil {
		return o.DefaultVisibleFields
	}
	return []string{}
}

// GetFieldOrdering mirrors getFieldOrdering.
func GetFieldOrdering(entityName string, override *EntityOverride) map[string]float64 {
	if o := resolveOverride(entityName, override); o != nil && o.DefaultFieldOrdering != nil {
		return o.DefaultFieldOrdering
	}
	return map[string]float64{}
}

// GetFieldRenderModes mirrors getFieldRenderModes.
func GetFieldRenderModes(entityName string, override *EntityOverride) map[string]string {
	if o := resolveOverride(entityName, override); o != nil && o.FieldRenderModes != nil {
		return o.FieldRenderModes
	}
	return map[string]string{}
}

// GetFieldMetadata mirrors getFieldMetadata.
func GetFieldMetadata(entityName string, override *EntityOverride) map[string]*dal.Record {
	if o := resolveOverride(entityName, override); o != nil && o.FieldMetadata != nil {
		return o.FieldMetadata
	}
	return map[string]*dal.Record{}
}

// GetAdditionalTypes mirrors getAdditionalTypes.
func GetAdditionalTypes(entityName string, override *EntityOverride) []string {
	if o := resolveOverride(entityName, override); o != nil && o.AdditionalTypes != nil {
		return o.AdditionalTypes
	}
	return []string{}
}

// GetNonFilterableFields mirrors getNonFilterableFields.
func GetNonFilterableFields(entityName string, override *EntityOverride) []string {
	if o := resolveOverride(entityName, override); o != nil && o.NonFilterableFields != nil {
		return o.NonFilterableFields
	}
	return []string{}
}

// GetNonSortableFields mirrors getNonSortableFields.
func GetNonSortableFields(entityName string, override *EntityOverride) []string {
	if o := resolveOverride(entityName, override); o != nil && o.NonSortableFields != nil {
		return o.NonSortableFields
	}
	return []string{}
}

// RouteToEntityMap mirrors ROUTE_TO_ENTITY_MAP.
var RouteToEntityMap = map[string]string{
	"orders":         "Order",
	"products":       "Product",
	"customers":      "Customer",
	"users":          "User",
	"regions":        "Region",
	"sales-channels": "SalesChannel",
}

// EntityNameFromRoute mirrors entityNameFromRoute ("" when unknown).
func EntityNameFromRoute(routeName string) string {
	return RouteToEntityMap[routeName]
}

// normaliseOption returns an option value as decoded JSON (objects as
// ordered records, arrays as []any).
func normaliseOption(v any) any {
	data, err := dal.MarshalValue(v)
	if err != nil {
		return nil
	}
	out, err := dal.DecodeOrderedJSON(data)
	if err != nil {
		return nil
	}
	return out
}

func optionStrings(v any) []string {
	list, ok := v.([]any)
	if !ok {
		return nil
	}
	out := []string{}
	for _, x := range list {
		if s, ok := x.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func optionString(v any) string {
	s, _ := v.(string)
	return s
}

func optionRecord(v any) *dal.Record {
	r, _ := v.(*dal.Record)
	return r
}

// computedColumnFromOption decodes a ComputedColumnDefinition option.
func computedColumnFromOption(r *dal.Record) *ComputedColumnDefinition {
	c := &ComputedColumnDefinition{}
	for _, k := range r.Keys() {
		v, _ := r.Get(k)
		switch k {
		case "id":
			c.ID = optionString(v)
		case "name":
			c.Name = optionString(v)
		case "renderMode":
			c.RenderMode = optionString(v)
		case "requiredFields":
			c.RequiredFields = optionStrings(v)
		case "optionalFields":
			c.OptionalFields = optionStrings(v)
		case "defaultVisible":
			c.DefaultVisible, _ = v.(bool)
		case "description":
			c.Description = optionString(v)
		case "category":
			c.Category = optionString(v)
		case "metadata":
			c.Metadata = optionRecord(v)
		case "context":
			c.Context = optionString(v)
		case "filter":
			c.Filter = optionRecord(v)
		case "dataType":
			c.DataType = optionString(v)
		case "sortable":
			c.Sortable, _ = v.(bool)
		}
	}
	return c
}

// SpreadEntityOverride mirrors `{ ...existing, ...override }` for an
// override given in the module options (settings entityOverrides): the
// keys the option defines replace the existing values. It also returns the
// computed columns the option defines (override.computedColumns).
func SpreadEntityOverride(existing *EntityOverride, option any) (EntityOverride, []*ComputedColumnDefinition) {
	out := EntityOverride{}
	if existing != nil {
		out = *existing
	}
	r, ok := normaliseOption(option).(*dal.Record)
	if !ok {
		return out, nil
	}
	var computed []*ComputedColumnDefinition
	for _, k := range r.Keys() {
		v, _ := r.Get(k)
		switch k {
		case "excludeFields":
			out.ExcludeFields = optionStrings(v)
		case "excludeSuffixes":
			out.ExcludeSuffixes = optionStrings(v)
		case "excludePrefixes":
			out.ExcludePrefixes = optionStrings(v)
		case "defaultVisibleFields":
			out.DefaultVisibleFields = optionStrings(v)
		case "defaultFieldOrdering":
			out.DefaultFieldOrdering = nil
			if m := optionRecord(v); m != nil {
				out.DefaultFieldOrdering = map[string]float64{}
				for _, f := range m.Keys() {
					n, _ := m.Get(f)
					if x, ok := n.(float64); ok {
						out.DefaultFieldOrdering[f] = x
					}
				}
			}
		case "fieldRenderModes":
			out.FieldRenderModes = nil
			if m := optionRecord(v); m != nil {
				out.FieldRenderModes = map[string]string{}
				for _, f := range m.Keys() {
					s, _ := m.Get(f)
					out.FieldRenderModes[f] = optionString(s)
				}
			}
		case "fieldMetadata":
			out.FieldMetadata = nil
			if m := optionRecord(v); m != nil {
				out.FieldMetadata = map[string]*dal.Record{}
				for _, f := range m.Keys() {
					x, _ := m.Get(f)
					out.FieldMetadata[f] = optionRecord(x)
				}
			}
		case "additionalTypes":
			out.AdditionalTypes = optionStrings(v)
		case "nonFilterableFields":
			out.NonFilterableFields = optionStrings(v)
		case "nonSortableFields":
			out.NonSortableFields = optionStrings(v)
		case "computedColumns":
			out.ComputedColumns = nil
			if list, ok := v.([]any); ok {
				out.ComputedColumns = []*ComputedColumnDefinition{}
				for _, x := range list {
					if cr := optionRecord(x); cr != nil {
						out.ComputedColumns = append(out.ComputedColumns, computedColumnFromOption(cr))
					}
				}
			}
			computed = out.ComputedColumns
		}
	}
	return out, computed
}
