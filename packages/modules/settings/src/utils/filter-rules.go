package utils

import (
	"regexp"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// nonFilterableTypes mirrors NON_FILTERABLE_TYPES.
var nonFilterableTypes = []string{"object"}

// nonFilterablePatterns mirrors NON_FILTERABLE_PATTERNS.
var nonFilterablePatterns = []*regexp.Regexp{
	regexp.MustCompile(`^raw_`),      // Raw data fields
	regexp.MustCompile(`^metadata$`), // Metadata JSON fields
	regexp.MustCompile(`_link$`),     // Link fields
}

// nonFilterableSemanticTypes mirrors NON_FILTERABLE_SEMANTIC_TYPES.
var nonFilterableSemanticTypes = []string{"object", "json", "metadata"}

// filterOperatorsByType mirrors FILTER_OPERATORS_BY_TYPE.
var filterOperatorsByType = map[string][]string{
	"string":   {"eq", "ne", "contains", "startsWith", "endsWith", "in", "nin"},
	"number":   {"eq", "ne", "gt", "gte", "lt", "lte", "in", "nin"},
	"boolean":  {"eq"},
	"date":     {"eq", "ne", "gt", "gte", "lt", "lte"},
	"currency": {"eq", "ne", "gt", "gte", "lt", "lte"},
	"enum":     {"eq", "ne", "in", "nin"},
	"object":   {},
}

func includes(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// IsFilterable mirrors isFilterable: whether a field is filterable from
// its name, data type, semantic type and whether it is computed.
func IsFilterable(fieldName, dataType string, isComputed bool, semanticType string) bool {
	if isComputed {
		return false
	}
	if includes(nonFilterableTypes, dataType) {
		return false
	}
	for _, p := range nonFilterablePatterns {
		if p.MatchString(fieldName) {
			return false
		}
	}
	if semanticType != "" && includes(nonFilterableSemanticTypes, semanticType) {
		return false
	}
	return true
}

// GetFilterOperators mirrors getFilterOperators.
func GetFilterOperators(dataType string) []string {
	if ops, ok := filterOperatorsByType[dataType]; ok {
		return ops
	}
	return []string{}
}

// BuildFilterConfig mirrors buildFilterConfig: the column filter
// configuration { enabled, operators?, enumValues? } (undefined keys
// omitted). enumValues nil means undefined.
func BuildFilterConfig(fieldName, dataType string, isComputed bool, semanticType string, enumValues []string) *dal.Record {
	out := dal.NewRecord()
	if !IsFilterable(fieldName, dataType, isComputed, semanticType) {
		out.Set("enabled", false)
		return out
	}
	out.Set("enabled", true)
	if ops := GetFilterOperators(dataType); len(ops) > 0 {
		out.Set("operators", ops)
	}
	if dataType == "enum" && enumValues != nil {
		out.Set("enumValues", enumValues)
	}
	return out
}

// FieldFilterRules mirrors FieldFilterRules.
type FieldFilterRules struct {
	ExcludeSuffixes []string
	ExcludePrefixes []string
	ExcludeFields   []string
}

// DefaultFieldFilterRules mirrors DEFAULT_FIELD_FILTER_RULES.
var DefaultFieldFilterRules = FieldFilterRules{
	ExcludeSuffixes: []string{"_link"},
	ExcludePrefixes: []string{"raw_"},
	ExcludeFields:   []string{},
}

// ShouldExcludeField mirrors shouldExcludeField.
func ShouldExcludeField(fieldName string, rules FieldFilterRules) bool {
	for _, suffix := range rules.ExcludeSuffixes {
		if strings.HasSuffix(fieldName, suffix) {
			return true
		}
	}
	for _, prefix := range rules.ExcludePrefixes {
		if strings.HasPrefix(fieldName, prefix) {
			return true
		}
	}
	return includes(rules.ExcludeFields, fieldName)
}
