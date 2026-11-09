package commonvalidators

import (
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

func stringOrStrings() *zod.Schema { return zod.Union(zod.String(), zod.Array(zod.String())) }

// ProductStatusEnum mirrors z.nativeEnum(ProductStatus).
func ProductStatusEnum() *zod.Schema {
	return zod.Enum("draft", "proposed", "published", "rejected")
}

// StoreGetProductParamsDirectFields mirrors StoreGetProductParamsDirectFields.
func StoreGetProductParamsDirectFields(operatorMap func() *zod.Schema) *zod.Schema {
	return zod.Object(
		zod.F("q", zod.String().Optional()),
		zod.F("id", stringOrStrings().Optional()),
		zod.F("title", stringOrStrings().Optional()),
		zod.F("handle", stringOrStrings().Optional()),
		zod.F("is_giftcard", BooleanString().Optional()),
		zod.F("category_id", stringOrStrings().Optional()),
		zod.F("option_id", stringOrStrings().Optional()),
		zod.F("option_value_id", stringOrStrings().Optional()),
		zod.F("external_id", stringOrStrings().Optional()),
		zod.F("collection_id", stringOrStrings().Optional()),
		zod.F("tag_id", stringOrStrings().Optional()),
		zod.F("type_id", stringOrStrings().Optional()),
		zod.F("created_at", operatorMap().Optional()),
		zod.F("updated_at", operatorMap().Optional()),
		zod.F("deleted_at", operatorMap().Optional()),
	)
}

// GetProductsParams mirrors GetProductsParams.
func GetProductsParams(operatorMap func() *zod.Schema) *zod.Schema {
	return zod.Object(zod.F("sales_channel_id", stringOrStrings().Optional())).Merge(StoreGetProductParamsDirectFields(operatorMap))
}

func isPresent(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case string:
		return t != ""
	case []any:
		return len(t) > 0
	case *dal.Record:
		return t.Len() > 0
	}
	return true
}

// TransformProductParams mirrors transformProductParams.
func TransformProductParams(v any) (any, error) {
	data, ok := v.(*dal.Record)
	if !ok {
		return v, nil
	}
	res := dal.NewRecord()
	for _, k := range data.Keys() {
		x, _ := data.Get(k)
		res.Set(k, x)
	}
	if x, _ := data.Get("tag_id"); isPresent(x) {
		r := dal.NewRecord()
		r.Set("id", x)
		res.Set("tags", r)
		res.Delete("tag_id")
	}
	if x, _ := data.Get("category_id"); isPresent(x) {
		r := dal.NewRecord()
		r.Set("id", x)
		res.Set("categories", r)
		res.Delete("category_id")
	}
	if x, _ := data.Get("option_id"); isPresent(x) {
		r := dal.NewRecord()
		r.Set("id", x)
		res.Set("options", r)
		res.Delete("option_id")
	}
	return res, nil
}
