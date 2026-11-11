package query

import (
	"context"
	"sort"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/utils/src/caching"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// This file mirrors @medusajs/utils/src/translations/apply-translations.ts.
// It lives with Query because it reads the translations through it (the
// utils package cannot import Query).

var translationExcludedKeys = map[string]bool{
	"id": true, "created_at": true, "updated_at": true, "deleted_at": true, "metadata": true,
}

// EnableTranslations turns on applyTranslations (the translation feature
// flag, with its module loaded).
func (q *Query) EnableTranslations() { q.translations = true }

// TranslationsEnabled reports whether translations are applied.
func (q *Query) TranslationsEnabled() bool { return q != nil && q.translations }

func translatableObject(v any) (*dal.Record, bool) {
	r, ok := v.(*dal.Record)
	return r, ok && r != nil
}

func listItems(v any) ([]any, bool) {
	switch t := v.(type) {
	case []any:
		return t, true
	case []*dal.Record:
		out := make([]any, len(t))
		for i, r := range t {
			out[i] = r
		}
		return out, true
	}
	return nil, false
}

// canApplyTranslationTo mirrors canApplyTranslationTo: an object with a
// truthy id.
func canApplyTranslationTo(r *dal.Record) bool {
	id, ok := r.Get("id")
	if !ok || id == nil || id == false || id == "" {
		return false
	}
	if f, isNum := id.(float64); isNum && f == 0 {
		return false
	}
	return true
}

func gatherTranslationIDs(v any, ids map[string]bool, order *[]string) {
	r, ok := translatableObject(v)
	if !ok {
		return
	}
	if id, ok := r.Get("id"); ok {
		if s, isStr := id.(string); isStr && s != "" && !ids[s] {
			ids[s] = true
			*order = append(*order, s)
		}
	}
	for _, k := range r.Keys() {
		value, _ := r.Get(k)
		if items, isList := listItems(value); isList {
			for _, item := range items {
				if item != nil {
					gatherTranslationIDs(item, ids, order)
				}
			}
		} else if _, isObj := translatableObject(value); isObj {
			gatherTranslationIDs(value, ids, order)
		}
	}
}

// jsTypeOf mirrors typeof for JSON-like values.
func jsTypeOf(v any) string {
	switch v.(type) {
	case nil:
		return "object"
	case string:
		return "string"
	case bool:
		return "boolean"
	case float64, float32, int, int64, int32:
		return "number"
	}
	return "object"
}

func isEmptyStringTranslation(v any) bool {
	s, ok := v.(string)
	return ok && strings.TrimSpace(s) == ""
}

func applyTranslation(r *dal.Record, byID map[string]*dal.Record) {
	idv, _ := r.Get("id")
	id, _ := idv.(string)
	translation, hasTranslation := byID[id]
	for _, key := range r.Keys() {
		if translationExcludedKeys[key] {
			continue
		}
		value, _ := r.Get(key)
		if hasTranslation {
			if tv, ok := translation.Get(key); ok && jsTypeOf(value) == jsTypeOf(tv) && !isEmptyStringTranslation(tv) {
				r.Set(key, tv)
				continue
			}
		}
		if items, isList := listItems(value); isList {
			for _, item := range items {
				if obj, ok := translatableObject(item); ok && canApplyTranslationTo(obj) {
					applyTranslation(obj, byID)
				}
			}
		} else if obj, ok := translatableObject(value); ok && canApplyTranslationTo(obj) {
			applyTranslation(obj, byID)
		}
	}
}

// ApplyTranslations mirrors applyTranslations: the string fields of the
// objects (and of the nested objects with an id) are replaced by their
// translation in the locale.
func (q *Query) ApplyTranslations(ctx context.Context, locale string, objects []any) error {
	if !q.TranslationsEnabled() || locale == "" {
		return nil
	}
	var present []any
	for _, o := range objects {
		if o != nil {
			present = append(present, o)
		}
	}
	if len(present) == 0 {
		return nil
	}
	ids := map[string]bool{}
	var order []string
	for _, o := range present {
		gatherTranslationIDs(o, ids, &order)
	}
	const batchSize = 250
	byID := map[string]*dal.Record{}
	for start := 0; start < len(order); start += batchSize {
		end := start + batchSize
		if end > len(order) {
			end = len(order)
		}
		batch := append([]string{}, order[start:end]...)
		sort.Strings(batch)
		refs := make([]any, len(batch))
		for i, id := range batch {
			refs[i] = id
		}
		filters := dal.NewRecord()
		filters.Set("reference_id", refs)
		filters.Set("locale_code", locale)
		pagination := dal.NewRecord()
		pagination.Set("take", float64(batchSize))
		res, err := q.Graph(ctx, GraphInput{Entity: "translations", Fields: []string{"translations", "reference_id"}, Filters: filters, Pagination: pagination}, joiner.Options{Cache: caching.Enabled()})
		if err != nil {
			return err
		}
		for _, row := range res.Data {
			t, ok := row.(*dal.Record)
			if !ok {
				continue
			}
			refv, _ := t.Get("reference_id")
			ref, _ := refv.(string)
			tr, _ := t.Get("translations")
			trRec, _ := tr.(*dal.Record)
			if trRec == nil {
				trRec = dal.NewRecord()
			}
			byID[ref] = trRec
		}
	}
	for _, o := range present {
		if r, ok := translatableObject(o); ok {
			applyTranslation(r, byID)
		}
	}
	return nil
}
