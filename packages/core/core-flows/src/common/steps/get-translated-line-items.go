package steps

import (
	"strings"

	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// GetTranslatedLineItemsStepInput mirrors GetTranslatedLineItemsStepInput.
type GetTranslatedLineItemsStepInput struct {
	Items    []*dal.Record
	Variants []*dal.Record
	Locale   string
}

// cloneTree copies the records and arrays of a value (step data is a copy
// upstream: translating the variants must not change the caller's).
func cloneTree(v any) any {
	switch t := v.(type) {
	case *dal.Record:
		if t == nil {
			return t
		}
		n := dal.NewRecord()
		for _, k := range t.Keys() {
			x, _ := t.Get(k)
			n.Set(k, cloneTree(x))
		}
		return n
	case []any:
		out := make([]any, len(t))
		for i, x := range t {
			out[i] = cloneTree(x)
		}
		return out
	case []*dal.Record:
		out := make([]any, len(t))
		for i, x := range t {
			out[i] = cloneTree(x)
		}
		return out
	}
	return v
}

var translatableItemPropPrefixes = []string{"variant_", "product_", "type_", "collection_"}

func translatableItemEntity(prefix string, variant *dal.Record) *dal.Record {
	rec := func(r *dal.Record, key string) *dal.Record {
		if r == nil {
			return nil
		}
		v, _ := r.Get(key)
		x, _ := v.(*dal.Record)
		return x
	}
	switch prefix {
	case "variant_":
		return variant
	case "product_":
		return rec(variant, "product")
	case "type_":
		return rec(rec(variant, "product"), "type")
	case "collection_":
		return rec(rec(variant, "product"), "collection")
	}
	return nil
}

func jsTypeOf(v any, present bool) string {
	if !present {
		return "undefined"
	}
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

// ApplyTranslationsToItems mirrors applyTranslationsToItems: the
// variant_/product_/type_/collection_ fields of the line items take the
// (translated) values of their variant, product, type and collection, and
// the title the product title.
func ApplyTranslationsToItems(items []*dal.Record, variants []*dal.Record) []*dal.Record {
	byID := map[any]*dal.Record{}
	for _, v := range variants {
		if v == nil {
			continue
		}
		id, _ := v.Get("id")
		byID[id] = v
	}
	for _, item := range items {
		variantID, _ := item.Get("variant_id")
		if variantID == nil || variantID == "" {
			continue
		}
		variant, ok := byID[variantID]
		if !ok {
			continue
		}
		for _, key := range item.Keys() {
			for _, prefix := range translatableItemPropPrefixes {
				if !strings.HasPrefix(key, prefix) {
					continue
				}
				entity := translatableItemEntity(prefix, variant)
				if entity == nil {
					break
				}
				translationKey := strings.Replace(key, prefix, "", 1)
				current, hasCurrent := item.Get(key)
				translated, hasTranslated := entity.Get(translationKey)
				if jsTypeOf(current, hasCurrent) == jsTypeOf(translated, hasTranslated) {
					if hasTranslated {
						item.Set(key, translated)
					}
				}
			}
			if pt, ok := item.Get("product_title"); ok && pt != nil {
				item.Set("title", pt)
			} else {
				t, _ := item.Get("title")
				item.Set("title", t)
			}
		}
	}
	return items
}

// GetTranslatedLineItemsStep mirrors getTranslatedLineItemsStep.
var GetTranslatedLineItemsStep = composer.CreateStep("get-translated-line-items",
	func(ctx composer.StepExecutionContext, in GetTranslatedLineItemsStepInput) (composer.StepResponse[[]*dal.Record], error) {
		items := in.Items
		if items == nil {
			items = []*dal.Record{}
		}
		q, _ := ctx.Container.Resolve(common.ContainerRegistrationKeys.QUERY).(*query.Query)
		if !q.TranslationsEnabled() || in.Locale == "" || len(items) == 0 {
			return composer.NewStepResponse(items), nil
		}
		variants := make([]*dal.Record, 0, len(in.Variants))
		objects := make([]any, 0, len(in.Variants))
		for _, v := range in.Variants {
			c, _ := cloneTree(v).(*dal.Record)
			variants = append(variants, c)
			objects = append(objects, c)
		}
		if err := q.ApplyTranslations(ctx.Context, in.Locale, objects); err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		translated := make([]*dal.Record, len(items))
		for i, it := range items {
			c := dal.NewRecord()
			for _, k := range it.Keys() {
				v, _ := it.Get(k)
				c.Set(k, v)
			}
			translated[i] = c
		}
		return composer.NewStepResponse(ApplyTranslationsToItems(translated, variants)), nil
	}, nil)
