// Package utils mirrors @medusajs/translation/src/utils.
package utils

import (
	"regexp"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// ComputeTranslatedFieldCount mirrors computeTranslatedFieldCount: the
// translatable fields holding a value other than null, "" and "null".
func ComputeTranslatedFieldCount(translations *dal.Record, translatableFields []string) int {
	if translations == nil || len(translatableFields) == 0 {
		return 0
	}
	n := 0
	for _, f := range translatableFields {
		v, ok := translations.Get(f)
		if !ok || v == nil || v == "" || v == "null" {
			continue
		}
		n++
	}
	return n
}

// FilterTranslationFields mirrors filterTranslationFields: the translations
// of each record narrowed to the translatable fields of its reference ({}
// without any). The translations key is assigned, so it is added (last)
// when it was not selected.
func FilterTranslationFields(translations []*dal.Record, config map[string][]string) []*dal.Record {
	for _, t := range translations {
		ref, _ := t.Get("reference")
		refStr, _ := ref.(string)
		allowed := config[refStr]
		filtered := dal.NewRecord()
		if len(allowed) > 0 {
			current, _ := t.Get("translations")
			if cur, ok := current.(*dal.Record); ok {
				for _, f := range allowed {
					if v, ok := cur.Get(f); ok {
						filtered.Set(f, v)
					}
				}
			}
		}
		t.Set("translations", filtered)
	}
	return translations
}

var (
	snakeLowerUpper = regexp.MustCompile(`([a-z])([A-Z])`)
	snakeNonAlnum   = regexp.MustCompile(`[^a-zA-Z0-9]+`)
	snakeTrim       = regexp.MustCompile(`^_+|_+$`)
)

// ToSnakeCase mirrors toSnakeCase.
func ToSnakeCase(s string) string {
	s = snakeLowerUpper.ReplaceAllString(s, "${1}_${2}")
	s = snakeNonAlnum.ReplaceAllString(s, "_")
	s = snakeTrim.ReplaceAllString(s, "")
	return strings.ToLower(s)
}
