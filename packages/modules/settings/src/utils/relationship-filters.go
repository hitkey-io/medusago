package utils

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/utils/src/graphql"
)

// Relationship filter configuration utilities: dropdown filters for
// relationship fields (relationship-filters.ts).

// RelationshipFilterDefinition mirrors RelationshipFilterDefinition.
type RelationshipFilterDefinition struct {
	// Field is the relationship field (e.g. "sales_channels").
	Field string
	// RelatedEntity is the related entity name (e.g. "SalesChannel").
	RelatedEntity string
	// ValueField is the field used as filter value (e.g. "id").
	ValueField string
	// DisplayField is the field displayed in the dropdown (e.g. "name").
	DisplayField string
	// Multiple is whether multiple selection is allowed.
	Multiple bool
	// Endpoint is the API endpoint the options are fetched from ("" =
	// inferred).
	Endpoint string
	// FilterKey is the key used to filter the relationship ("" = the
	// column field).
	FilterKey string
}

// RelationshipFilterConfig mirrors RelationshipFilterConfig (the column
// metadata shape).
type RelationshipFilterConfig struct {
	Entity       string
	ValueField   string
	DisplayField string
	Multiple     bool
	Endpoint     string
	// FilterKey "" means undefined (omitted).
	FilterKey string
}

// ToRecord returns the config as the response object (filter_key omitted
// when undefined).
func (c *RelationshipFilterConfig) ToRecord() *dal.Record {
	out := dal.NewRecord()
	out.Set("entity", c.Entity)
	out.Set("value_field", c.ValueField)
	out.Set("display_field", c.DisplayField)
	out.Set("multiple", c.Multiple)
	out.Set("endpoint", c.Endpoint)
	if c.FilterKey != "" {
		out.Set("filter_key", c.FilterKey)
	}
	return out
}

// EntityEndpointMap mirrors ENTITY_ENDPOINT_MAP: the entities whose admin
// endpoint does not follow /admin/{kebab-case-plural}.
var EntityEndpointMap = map[string]string{
	// ProductCollection uses /admin/collections instead of /admin/product-collections
	"ProductCollection": "/admin/collections",
}

// InferOptionsEndpoint mirrors inferOptionsEndpoint.
func InferOptionsEndpoint(entityName string) string {
	if e := EntityEndpointMap[entityName]; e != "" {
		return e
	}
	return "/admin/" + ToKebabCasePlural(entityName)
}

// RelationshipFilterOverrides mirrors RELATIONSHIP_FILTER_OVERRIDES.
var RelationshipFilterOverrides = map[string][]RelationshipFilterDefinition{
	"Product": {
		{Field: "sales_channels", RelatedEntity: "SalesChannel", ValueField: "id", DisplayField: "name", Multiple: true, FilterKey: "sales_channel_id"},
		{Field: "collection", RelatedEntity: "ProductCollection", ValueField: "id", DisplayField: "title", Multiple: true, FilterKey: "collection_id"},
		{Field: "type", RelatedEntity: "ProductType", ValueField: "id", DisplayField: "value", Multiple: true, FilterKey: "type_id"},
		{Field: "tags", RelatedEntity: "ProductTag", ValueField: "id", DisplayField: "value", Multiple: true, FilterKey: "tag_id"},
		{Field: "categories", RelatedEntity: "ProductCategory", ValueField: "id", DisplayField: "name", Multiple: true, FilterKey: "category_id"},
	},
	"Order": {
		{Field: "region", RelatedEntity: "Region", ValueField: "id", DisplayField: "name", Multiple: true, FilterKey: "region_id"},
		{Field: "customer", RelatedEntity: "Customer", ValueField: "id", DisplayField: "email", Multiple: true, FilterKey: "customer_id"},
		{Field: "sales_channel", RelatedEntity: "SalesChannel", ValueField: "id", DisplayField: "name", Multiple: true, FilterKey: "sales_channel_id"},
	},
	"Customer": {
		{Field: "groups", RelatedEntity: "CustomerGroup", ValueField: "id", DisplayField: "name", Multiple: true},
	},
	"StockLocation": {
		{Field: "sales_channels", RelatedEntity: "SalesChannel", ValueField: "id", DisplayField: "name", Multiple: true, FilterKey: "sales_channel_id"},
		{Field: "address", RelatedEntity: "StockLocationAddress", ValueField: "id", DisplayField: "", Multiple: false, FilterKey: "address_id"},
	},
	"Promotion": {
		{Field: "campaign", RelatedEntity: "Campaign", ValueField: "id", DisplayField: "name", Multiple: true, FilterKey: "campaign_id"},
	},
	"ReservationItem": {
		{Field: "inventory_item", RelatedEntity: "InventoryItem", ValueField: "id", DisplayField: "sku", Multiple: true, FilterKey: "inventory_item_id"},
	},
	"CustomerGroup": {
		{Field: "customers", RelatedEntity: "Customer", ValueField: "id", DisplayField: "email", Multiple: true, FilterKey: "customers"},
	},
}

// preferredDisplayFields mirrors PREFERRED_DISPLAY_FIELDS (by priority).
var preferredDisplayFields = []string{"name", "title", "value", "label", "email", "display_id", "display_name"}

// inferDisplayField mirrors inferDisplayField: the first preferred String
// field of the related type, else its first String field, else "name".
func inferDisplayField(relatedEntityName string, schema *graphql.Schema) string {
	const defaultField = "name"
	if schema == nil {
		return defaultField
	}
	related := schema.Type(relatedEntityName)
	if related == nil || related.Kind != graphql.KindObject {
		return defaultField
	}
	var stringFields []string
	for _, f := range related.Fields {
		u := GetUnderlyingType(schema, f.Type)
		if u.IsScalar() && u.Name == "String" {
			stringFields = append(stringFields, f.Name)
		}
	}
	for _, pf := range preferredDisplayFields {
		if includes(stringFields, pf) {
			return pf
		}
	}
	if len(stringFields) > 0 {
		return stringFields[0]
	}
	return defaultField
}

// GetRelationshipFilterConfig mirrors getRelationshipFilterConfig: the
// predefined configuration of the field, else one with an inferred display
// field.
func GetRelationshipFilterConfig(entityName, fieldName, relatedEntityName string, isMultiple bool, schema *graphql.Schema) *RelationshipFilterConfig {
	for _, o := range RelationshipFilterOverrides[entityName] {
		if o.Field != fieldName {
			continue
		}
		endpoint := o.Endpoint
		if endpoint == "" {
			endpoint = InferOptionsEndpoint(o.RelatedEntity)
		}
		return &RelationshipFilterConfig{
			Entity:       o.RelatedEntity,
			ValueField:   o.ValueField,
			DisplayField: o.DisplayField,
			Multiple:     o.Multiple,
			Endpoint:     endpoint,
			FilterKey:    o.FilterKey,
		}
	}
	return &RelationshipFilterConfig{
		Entity:       relatedEntityName,
		ValueField:   "id",
		DisplayField: inferDisplayField(relatedEntityName, schema),
		Multiple:     isMultiple,
		Endpoint:     InferOptionsEndpoint(relatedEntityName),
	}
}
