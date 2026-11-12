package utils

import (
	"regexp"
	"sort"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/utils/src/graphql"
)

// PropertyLabel mirrors PropertyLabel: a custom column name.
type PropertyLabel struct {
	ID          string
	Entity      string
	Property    string
	Label       string
	Description *string
}

// getEnumValues mirrors getEnumValues: the real (stored) values of an
// enum, from the @enumValue(value: "...") directive of DML generated enums,
// else the member value.
func getEnumValues(enumType *graphql.Type) []string {
	out := make([]string, 0, len(enumType.Values))
	for _, v := range enumType.Values {
		value := v
		for _, d := range enumType.ValueDirectives[v] {
			if d.Name != "enumValue" {
				continue
			}
			for _, a := range d.Arguments {
				if a.Name == "value" && a.Value != nil && a.Value.Kind == graphql.ValueString {
					value = a.Value.Value
				}
				if a.Name == "value" {
					break
				}
			}
			break
		}
		out = append(out, value)
	}
	return out
}

var (
	upperCaseRe     = regexp.MustCompile(`([A-Z])`)
	leadingSpacesRe = regexp.MustCompile(`^\s+`)
)

// formatFieldName mirrors formatFieldName: a field name as a display name
// ("sales_channel" -> "Sales Channel").
func formatFieldName(fieldName string) string {
	s := strings.ReplaceAll(fieldName, "_", " ")
	s = upperCaseRe.ReplaceAllString(s, " $1")
	s = leadingSpacesRe.ReplaceAllString(s, "")
	words := strings.Split(s, " ")
	for i, w := range words {
		if w == "" {
			continue
		}
		words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
	}
	return strings.Join(words, " ")
}

// semanticTypeToCategory mirrors semanticTypeToCategory.
func semanticTypeToCategory(semanticType string) string {
	switch semanticType {
	case "identifier":
		return "identifier"
	case "timestamp":
		return "timestamp"
	case "status":
		return "status"
	case "currency", "count":
		return "metric"
	}
	return "field"
}

// columnConfig is what processEntityType needs from the entity override.
type columnConfig struct {
	filterRules          FieldFilterRules
	defaultVisibleFields []string
	defaultFieldOrdering map[string]float64
	fieldRenderModes     map[string]string
	fieldMetadata        map[string]*dal.Record
	nonFilterableFields  []string
	nonSortableFields    []string
	propertyLabels       map[string]PropertyLabel
}

// order mirrors `defaultFieldOrdering[path] || fallback`.
func (c *columnConfig) order(path string, fallback float64) float64 {
	if v := c.defaultFieldOrdering[path]; v != 0 {
		return v
	}
	return fallback
}

// label returns the property label of a path.
func (c *columnConfig) label(path string) (PropertyLabel, bool) {
	l, ok := c.propertyLabels[path]
	return l, ok
}

// setLabelled sets name and description from the property label when it
// has them (label?.label || name, label?.description || undefined).
func setLabelled(col *dal.Record, label PropertyLabel, hasLabel bool, name string) {
	if hasLabel && label.Label != "" {
		col.Set("name", label.Label)
	} else {
		col.Set("name", name)
	}
	if hasLabel && label.Description != nil && *label.Description != "" {
		col.Set("description", *label.Description)
	}
}

func setLabelTail(col *dal.Record, entity *DiscoveredEntity, label PropertyLabel, hasLabel bool) {
	col.Set("source", rec("module", entity.Module, "entity", entity.Name))
	col.Set("custom_label", hasLabel)
	if hasLabel {
		col.Set("label_id", label.ID)
	}
}

// GenerateEntityColumns mirrors generateEntityColumns: the columns of an
// entity (its schema fields, the fields of its additional types, its
// computed columns) sorted by default order. Nil means null (discovery not
// initialised, unknown entity).
func GenerateEntityColumns(discovery *EntityDiscoveryService, entityKey string, propertyLabels map[string]PropertyLabel, customOverride *EntityOverride) []*dal.Record {
	if !discovery.IsInitialized() {
		return nil
	}
	entity := discovery.GetEntity(entityKey)
	if entity == nil {
		return nil
	}

	override := customOverride
	if override == nil {
		override = GetEntityOverride(entity.Name)
	}
	cfg := &columnConfig{
		filterRules:          GetFieldFilterRules(entity.Name, override),
		defaultVisibleFields: GetDefaultVisibleFields(entity.Name, override),
		defaultFieldOrdering: GetFieldOrdering(entity.Name, override),
		fieldRenderModes:     GetFieldRenderModes(entity.Name, override),
		fieldMetadata:        GetFieldMetadata(entity.Name, override),
		nonFilterableFields:  GetNonFilterableFields(entity.Name, override),
		nonSortableFields:    GetNonSortableFields(entity.Name, override),
		propertyLabels:       propertyLabels,
	}
	additionalTypes := GetAdditionalTypes(entity.Name, override)

	schema := discovery.GetSchemaTypeMap()
	columns := []*dal.Record{}
	processed := map[string]bool{}

	// Process main entity type
	processEntityType(schema, entity.GraphqlType, entity, &columns, processed, cfg, "")

	// Process additional types (e.g., OrderDetail for Order)
	for _, additional := range additionalTypes {
		if schema.Type(additional) != nil {
			processEntityType(schema, additional, entity, &columns, processed, cfg, "")
		}
	}

	// Add computed columns
	for _, computed := range GetComputedColumnRegistry().Get(entity.Name) {
		columnID := computed.ID
		if processed[columnID] {
			continue
		}
		label, hasLabel := cfg.label(columnID)

		col := dal.NewRecord()
		col.Set("id", columnID)
		if hasLabel && label.Label != "" {
			col.Set("name", label.Label)
		} else {
			col.Set("name", computed.Name)
		}
		if hasLabel && label.Description != nil && *label.Description != "" {
			col.Set("description", *label.Description)
		} else if computed.Description != "" {
			col.Set("description", computed.Description)
		}
		col.Set("field", columnID)
		col.Set("sortable", computed.Sortable)
		col.Set("hideable", true)
		col.Set("default_visible", computed.DefaultVisible || includes(cfg.defaultVisibleFields, columnID))
		dataType := computed.DataType
		if dataType == "" {
			dataType = "string"
		}
		col.Set("data_type", dataType)
		if computed.RenderMode != "" {
			col.Set("semantic_type", "computed")
		} else {
			col.Set("semantic_type", "relationship")
		}
		context := computed.Context
		if context == "" {
			context = "display"
		}
		col.Set("context", context)
		// Only display columns carry a compute (render + required fields);
		// filter-only injected columns omit it.
		if computed.RenderMode != "" {
			required := computed.RequiredFields
			if required == nil {
				required = []string{}
			}
			optional := computed.OptionalFields
			if optional == nil {
				optional = []string{}
			}
			col.Set("computed", rec("type", computed.RenderMode, "required_fields", required, "optional_fields", optional))
		}
		if computed.Metadata != nil {
			col.Set("metadata", computed.Metadata)
		}
		if computed.RenderMode != "" {
			col.Set("render_mode", computed.RenderMode)
		}
		col.Set("default_order", cfg.order(columnID, 850))
		category := computed.Category
		if category == "" {
			category = "computed"
		}
		col.Set("category", category)
		if computed.Filter != nil {
			col.Set("filter", computed.Filter)
		} else {
			col.Set("filter", rec("enabled", false))
		}
		setLabelTail(col, entity, label, hasLabel)
		columns = append(columns, col)
	}

	// Sort columns by default_order
	sort.SliceStable(columns, func(i, j int) bool {
		return defaultOrder(columns[i]) < defaultOrder(columns[j])
	})
	return columns
}

// defaultOrder mirrors `a.default_order || 1000`.
func defaultOrder(col *dal.Record) float64 {
	if v, ok := col.Get("default_order"); ok {
		if f, ok := v.(float64); ok && f != 0 {
			return f
		}
	}
	return 1000
}

// processEntityType mirrors processEntityType: the columns of the fields
// of a GraphQL type (scalars and enums, the scalar fields of single
// relationships one level deep, filter columns of array relationships).
func processEntityType(schema *graphql.Schema, typeName string, entity *DiscoveredEntity, columns *[]*dal.Record, processed map[string]bool, cfg *columnConfig, parentPath string) {
	t := schema.Type(typeName)
	if t == nil || (t.Kind != graphql.KindObject && t.Kind != graphql.KindInterface && t.Kind != graphql.KindInput) {
		return
	}
	fields := t.Fields

	// Foreign keys made redundant by a single relationship of the same name
	// (e.g. collection_id -> collection.id). The relationship's `.id` column
	// covers both filtering and display, so drop the raw FK to avoid
	// duplicates.
	redundantForeignKeys := map[string]bool{}
	if parentPath == "" {
		for _, f := range fields {
			if IsSingleRelationship(schema, f.Type) && GetUnderlyingType(schema, f.Type).IsObject() {
				redundantForeignKeys[f.Name+"_id"] = true
			}
		}
	}

	for _, f := range fields {
		fieldName := f.Name
		fullPath := fieldName
		if parentPath != "" {
			fullPath = parentPath + "." + fieldName
		}

		// Skip if already processed
		if processed[fullPath] {
			continue
		}
		// Skip excluded fields
		if ShouldExcludeField(fieldName, cfg.filterRules) {
			continue
		}
		// Skip FKs covered by their relationship's `.id` column
		if redundantForeignKeys[fullPath] {
			continue
		}

		underlying := GetUnderlyingType(schema, f.Type)
		isArray := IsArrayField(f.Type)

		switch {
		// Handle scalar and enum types
		case underlying.IsScalar() || underlying.IsEnum():
			dataType := InferDataType(underlying, fieldName)
			renderMode, semanticType := InferRenderMode(fieldName, underlying)
			if m, ok := cfg.fieldRenderModes[fullPath]; ok {
				renderMode = m
			}
			label, hasLabel := cfg.label(fullPath)
			processed[fullPath] = true

			var filter *dal.Record
			if includes(cfg.nonFilterableFields, fullPath) {
				filter = rec("enabled", false)
			} else {
				var enumValues []string
				if underlying.IsEnum() {
					enumValues = getEnumValues(underlying.Type)
				}
				filter = BuildFilterConfig(fieldName, dataType, false, semanticType, enumValues)
			}

			col := dal.NewRecord()
			col.Set("id", fullPath)
			setLabelled(col, label, hasLabel, formatFieldName(fieldName))
			col.Set("field", fullPath)
			// Only top-level fields are sortable
			col.Set("sortable", parentPath == "" && dataType != "object" && !includes(cfg.nonSortableFields, fullPath))
			col.Set("hideable", true)
			col.Set("default_visible", includes(cfg.defaultVisibleFields, fullPath))
			col.Set("data_type", dataType)
			col.Set("semantic_type", semanticType)
			col.Set("context", "both")
			col.Set("render_mode", renderMode)
			if m := cfg.fieldMetadata[fullPath]; m != nil {
				col.Set("metadata", m)
			}
			col.Set("default_order", cfg.order(fullPath, 900))
			if parentPath != "" {
				col.Set("category", "relationship")
			} else {
				col.Set("category", semanticTypeToCategory(semanticType))
			}
			col.Set("filter", filter)
			setLabelTail(col, entity, label, hasLabel)
			*columns = append(*columns, col)

		// Handle single relationships (many-to-one, one-to-one)
		case IsSingleRelationship(schema, f.Type) && underlying.IsObject():
			// Process nested fields with dot notation (one level deep)
			if parentPath != "" {
				continue
			}
			relatedTypeName := underlying.Name
			for _, rf := range underlying.Type.Fields {
				relatedFieldName := rf.Name
				nestedPath := fieldName + "." + relatedFieldName
				nestedUnderlying := GetUnderlyingType(schema, rf.Type)

				// Only include scalar fields (not nested objects or arrays)
				if !nestedUnderlying.IsScalar() || processed[nestedPath] {
					continue
				}
				dataType := InferDataType(nestedUnderlying, relatedFieldName)
				renderMode, semanticType := InferRenderMode(relatedFieldName, nestedUnderlying)
				if m, ok := cfg.fieldRenderModes[nestedPath]; ok {
					renderMode = m
				}
				label, hasLabel := cfg.label(nestedPath)
				processed[nestedPath] = true

				var filter *dal.Record
				if includes(cfg.nonFilterableFields, nestedPath) {
					filter = rec("enabled", false)
				} else {
					var enumValues []string
					if nestedUnderlying.IsEnum() {
						enumValues = getEnumValues(nestedUnderlying.Type)
					}
					filter = BuildFilterConfig(relatedFieldName, dataType, false, semanticType, enumValues)
				}

				// Note: nested .id paths in nonFilterableFields produce
				// { enabled: false, relationship: {...} }; not currently
				// reachable.
				if relatedFieldName == "id" {
					if rel := GetRelationshipFilterConfig(entity.Name, fieldName, relatedTypeName, false, schema); rel != nil {
						filter.Set("relationship", rel.ToRecord())
					}
				}

				col := dal.NewRecord()
				col.Set("id", nestedPath)
				setLabelled(col, label, hasLabel, formatFieldName(fieldName)+" "+formatFieldName(relatedFieldName))
				col.Set("field", nestedPath)
				col.Set("sortable", false)
				col.Set("hideable", true)
				col.Set("default_visible", includes(cfg.defaultVisibleFields, nestedPath))
				col.Set("data_type", dataType)
				col.Set("semantic_type", semanticType)
				col.Set("context", "both")
				col.Set("render_mode", renderMode)
				if m := cfg.fieldMetadata[nestedPath]; m != nil {
					col.Set("metadata", m)
				}
				col.Set("default_order", cfg.order(nestedPath, 950))
				col.Set("category", "relationship")
				col.Set("filter", filter)
				setLabelTail(col, entity, label, hasLabel)
				*columns = append(*columns, col)
			}

		// Handle array relationships for relationship filter dropdowns
		case isArray && underlying.IsObject():
			if parentPath != "" {
				continue
			}
			// Add a relationship filter if this is a filterable entity
			rel := GetRelationshipFilterConfig(entity.Name, fieldName, underlying.Name, true, schema)
			if rel == nil {
				continue
			}
			label, hasLabel := cfg.label(fieldName)

			// Add as a virtual filterable column
			col := dal.NewRecord()
			col.Set("id", fieldName+"_filter")
			setLabelled(col, label, hasLabel, formatFieldName(fieldName))
			col.Set("field", fieldName+".id")
			col.Set("sortable", false)
			col.Set("hideable", true)
			col.Set("default_visible", false)
			col.Set("data_type", "string")
			col.Set("semantic_type", "relationship")
			col.Set("context", "filter")
			col.Set("render_mode", "text")
			col.Set("default_order", float64(980))
			col.Set("category", "relationship")
			col.Set("filter", rec("enabled", true, "operators", []string{"in"}, "relationship", rel.ToRecord()))
			setLabelTail(col, entity, label, hasLabel)
			*columns = append(*columns, col)
		}
	}
}
