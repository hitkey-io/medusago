package utils

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// LineItemFieldsForMerging mirrors lineItemFieldsForMerging.
var LineItemFieldsForMerging = []string{"id", "metadata", "variant_id", "quantity", "unit_price", "compare_at_unit_price", "is_custom_price"}

// DeepEqual mirrors deepEqualObj for record values.
func DeepEqual(a, b any) bool {
	ra, aok := a.(*dal.Record)
	rb, bok := b.(*dal.Record)
	if aok || bok {
		if !aok || !bok || ra == nil || rb == nil {
			return ra == nil && rb == nil && aok == bok
		}
		if ra.Len() != rb.Len() {
			return false
		}
		for _, k := range ra.Keys() {
			bv, ok := rb.Get(k)
			if !ok {
				return false
			}
			av, _ := ra.Get(k)
			if !DeepEqual(av, bv) {
				return false
			}
		}
		return true
	}
	la, aok := a.([]any)
	lb, bok := b.([]any)
	if aok || bok {
		if !aok || !bok || len(la) != len(lb) {
			return false
		}
		for i := range la {
			if !DeepEqual(la[i], lb[i]) {
				return false
			}
		}
		return true
	}
	return a == b
}

func metadataMatches(existing, newItem *dal.Record) bool {
	return (!IsPresent(Get(existing, "metadata")) && !IsPresent(Get(newItem, "metadata"))) || DeepEqual(Get(existing, "metadata"), Get(newItem, "metadata"))
}

// FindMatchingLineItem mirrors findMatchingLineItem (newItem carries
// metadata, unit_price and is_custom_price).
func FindMatchingLineItem(existing []*dal.Record, newItem *dal.Record) *dal.Record {
	for _, e := range existing {
		if Truthy(Get(newItem, "is_custom_price")) {
			if metadataMatches(e, newItem) && Get(newItem, "unit_price") == Get(e, "unit_price") {
				return e
			}
		} else if metadataMatches(e, newItem) && !Truthy(Get(e, "is_custom_price")) {
			return e
		}
	}
	return nil
}
