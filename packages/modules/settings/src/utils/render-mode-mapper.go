package utils

import (
	"regexp"
	"strings"
)

// Render mode mapping utilities for column generation: field names and
// GraphQL types mapped to render modes (render-mode-mapper.ts).

// typeToRenderMode mirrors TYPE_TO_RENDER_MODE.
var typeToRenderMode = map[string]string{
	// Primitive types
	"boolean": "boolean",
	"text":    "text",
	"number":  "number",

	// Medusa scalar types
	"dateTime": "datetime",
	"json":     "json",
	"id":       "id",

	// GraphQL scalars
	"String":   "text",
	"Int":      "number",
	"Float":    "number",
	"Boolean":  "boolean",
	"DateTime": "datetime",
	"JSON":     "json",
	"ID":       "id",
}

// fieldPatternOverride mirrors FieldPatternOverride.
type fieldPatternOverride struct {
	pattern      *regexp.Regexp
	renderMode   string
	semanticType string
}

func fpo(pattern, renderMode, semanticType string) fieldPatternOverride {
	return fieldPatternOverride{regexp.MustCompile(pattern), renderMode, semanticType}
}

// fieldPatternOverrides mirrors FIELD_PATTERN_OVERRIDES (first match wins).
var fieldPatternOverrides = []fieldPatternOverride{
	// Timestamps
	fpo(`_at$`, "datetime", "timestamp"),
	fpo(`^created_at$`, "datetime", "timestamp"),
	fpo(`^updated_at$`, "datetime", "timestamp"),
	fpo(`^deleted_at$`, "datetime", "timestamp"),
	fpo(`_date$`, "date", "date"),

	// Currency/amounts
	fpo(`total$`, "currency", "currency"),
	fpo(`amount$`, "currency", "currency"),
	fpo(`price$`, "currency", "currency"),
	fpo(`^subtotal$`, "currency", "currency"),
	fpo(`^tax_total$`, "currency", "currency"),
	fpo(`^shipping_total$`, "currency", "currency"),
	fpo(`^discount_total$`, "currency", "currency"),

	// Status fields
	fpo(`status$`, "status", "status"),
	fpo(`^state$`, "status", "status"),
	fpo(`^payment_status$`, "status", "status"),
	fpo(`^fulfillment_status$`, "status", "status"),

	// Contact info
	fpo(`^email$`, "email", "contact"),
	fpo(`_email$`, "email", "contact"),
	fpo(`^phone$`, "phone", "contact"),
	fpo(`_phone$`, "phone", "contact"),

	// Location
	fpo(`country_code$`, "country_code", "location"),
	fpo(`^currency_code$`, "text", "currency_code"),

	// URLs and images
	fpo(`^url$`, "url", "url"),
	fpo(`_url$`, "url", "url"),
	fpo(`^thumbnail$`, "image", "image"),
	fpo(`^avatar$`, "image", "image"),
	fpo(`_image$`, "image", "image"),

	// Identifiers
	fpo(`^id$`, "id", "identifier"),
	fpo(`^display_id$`, "display_id", "identifier"),
	fpo(`^custom_display_id$`, "display_id", "identifier"),
	fpo(`_id$`, "id", "identifier"),
	fpo(`^handle$`, "text", "identifier"),
	fpo(`^code$`, "text", "identifier"),

	// Counts
	fpo(`^(count|.*_count)$`, "number", "count"),
	fpo(`quantity$`, "number", "count"),

	// Booleans
	fpo(`^is_`, "boolean", "boolean"),
	fpo(`^has_`, "boolean", "boolean"),
	fpo(`^can_`, "boolean", "boolean"),

	// Metadata
	fpo(`^metadata$`, "json", "metadata"),
}

// InferRenderMode mirrors inferRenderMode: the render mode and semantic type
// of a field from its name (patterns first) and its GraphQL type.
func InferRenderMode(fieldName string, graphqlType NamedType) (renderMode, semanticType string) {
	for _, o := range fieldPatternOverrides {
		if o.pattern.MatchString(fieldName) {
			return o.renderMode, o.semanticType
		}
	}
	if graphqlType.IsEnum() {
		return "status", "enum"
	}
	if name := graphqlType.Name; name != "" {
		if mode, ok := typeToRenderMode[name]; ok {
			return mode, strings.ToLower(name)
		}
	}
	return "text", "string"
}

// DataTypeToRenderMode mirrors dataTypeToRenderMode.
func DataTypeToRenderMode(dataType string) string {
	switch dataType {
	case "string":
		return "text"
	case "number":
		return "number"
	case "boolean":
		return "boolean"
	case "date":
		return "datetime"
	case "currency":
		return "currency"
	case "enum":
		return "status"
	case "object":
		return "json"
	}
	return "text"
}

var (
	dateFieldRe     = regexp.MustCompile(`_at$`)
	timestampRe     = regexp.MustCompile(`^created_at$|^updated_at$|^deleted_at$`)
	currencyFieldRe = regexp.MustCompile(`total$|amount$|price$`)
	booleanFieldRe  = regexp.MustCompile(`^is_|^has_|^can_`)
	metadataFieldRe = regexp.MustCompile(`^metadata$`)
)

// InferDataType mirrors inferDataType: the column data type of a field from
// its name (patterns first) and its GraphQL type.
func InferDataType(graphqlType NamedType, fieldName string) string {
	if dateFieldRe.MatchString(fieldName) || timestampRe.MatchString(fieldName) {
		return "date"
	}
	if currencyFieldRe.MatchString(fieldName) {
		return "currency"
	}
	if graphqlType.IsEnum() {
		return "enum"
	}
	if booleanFieldRe.MatchString(fieldName) {
		return "boolean"
	}
	if metadataFieldRe.MatchString(fieldName) {
		return "object"
	}
	switch graphqlType.Name {
	case "String", "ID":
		return "string"
	case "Int", "Float":
		return "number"
	case "Boolean":
		return "boolean"
	case "DateTime":
		return "date"
	case "JSON":
		return "object"
	}
	return "string"
}
