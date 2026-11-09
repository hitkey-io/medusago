// Package products mirrors api/utils/middlewares/products.
package products

import (
	"context"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"math"

	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
)

var inventoryQuantityRequiredFields = []string{"id", "manage_inventory"}

// PrepareInventoryQuantityFields mirrors prepareInventoryQuantityFields.
func PrepareInventoryQuantityFields(fields []string, relation string) ([]string, bool) {
	prefix := ""
	if relation != "" {
		prefix = relation + "."
	}
	field := prefix + "inventory_quantity"
	with := false
	for _, f := range fields {
		if f == field {
			with = true
		}
	}
	if !with {
		return fields, false
	}
	seen := map[string]bool{}
	var out []string
	for _, f := range fields {
		if f != field && !seen[f] {
			seen[f] = true
			out = append(out, f)
		}
	}
	for _, r := range inventoryQuantityRequiredFields {
		if !seen[prefix+r] {
			seen[prefix+r] = true
			out = append(out, prefix+r)
		}
	}
	return out, true
}

func recordsOf(v any) []*dal.Record {
	switch t := v.(type) {
	case []any:
		var out []*dal.Record
		for _, x := range t {
			if r, ok := x.(*dal.Record); ok {
				out = append(out, r)
			}
		}
		return out
	case []*dal.Record:
		return t
	}
	return nil
}

// GetTotalVariantAvailability mirrors getTotalVariantAvailability: per
// variant, the min over its inventory items of floor(sum of available
// quantities / required quantity), nil without inventory items.
func GetTotalVariantAvailability(ctx context.Context, q *query.Query, variantIDs []any) (map[string]any, error) {
	return getVariantAvailability(ctx, q, variantIDs, "")
}

// GetVariantAvailability mirrors getVariantAvailability: only the levels of
// the sales channel's stock locations count.
func GetVariantAvailability(ctx context.Context, q *query.Query, variantIDs []any, salesChannelID string) (map[string]any, error) {
	return getVariantAvailability(ctx, q, variantIDs, salesChannelID)
}

func getVariantAvailability(ctx context.Context, q *query.Query, variantIDs []any, salesChannelID string) (map[string]any, error) {
	f := dal.NewRecord()
	f.Set("variant_id", variantIDs)
	res, err := q.Graph(ctx, query.GraphInput{
		Entity:  "product_variant_inventory_items",
		Fields:  []string{"variant_id", "required_quantity", "variant.manage_inventory", "variant.allow_backorder", "inventory.*", "inventory.location_levels.*"},
		Filters: f,
	}, joiner.Options{})
	if err != nil {
		return nil, err
	}
	locations := map[string]bool{}
	if salesChannelID != "" {
		lf := dal.NewRecord()
		lf.Set("sales_channel_id", salesChannelID)
		lr, err := q.Graph(ctx, query.GraphInput{Entity: "sales_channel_locations", Fields: []string{"stock_location_id"}, Filters: lf}, joiner.Options{})
		if err != nil {
			return nil, err
		}
		for _, row := range lr.Data {
			if r, ok := row.(*dal.Record); ok {
				v, _ := r.Get("stock_location_id")
				if s, ok := v.(string); ok {
					locations[s] = true
				}
			}
		}
	}
	byVariant := map[string][]*dal.Record{}
	for _, row := range res.Data {
		r, ok := row.(*dal.Record)
		if !ok {
			continue
		}
		vid, _ := r.Get("variant_id")
		s, _ := vid.(string)
		byVariant[s] = append(byVariant[s], r)
	}
	out := map[string]any{}
	for _, v := range variantIDs {
		id, _ := v.(string)
		links := byVariant[id]
		if len(links) == 0 {
			out[id] = nil
			continue
		}
		minQ := math.Inf(1)
		for _, link := range links {
			req, _ := link.Get("required_quantity")
			inv, _ := link.Get("inventory")
			invRec, _ := inv.(*dal.Record)
			sum := totals.Convert(0)
			if invRec != nil {
				lv, _ := invRec.Get("location_levels")
				for _, level := range recordsOf(lv) {
					if salesChannelID != "" {
						loc, _ := level.Get("location_id")
						if s, _ := loc.(string); !locations[s] {
							continue
						}
					}
					aq, _ := level.Get("available_quantity")
					if aq != nil {
						sum = totals.Add(sum, aq)
					}
				}
			}
			reqF := totals.Convert(req)
			sumF, _ := sum.Float64()
			rq, _ := reqF.Float64()
			q := math.Floor(sumF / rq)
			if q < minQ {
				minQ = q
			}
		}
		out[id] = minQ
	}
	return out, nil
}

// WrapVariantsWithInventoryQuantityForSalesChannel mirrors
// wrapVariantsWithInventoryQuantityForSalesChannel.
func WrapVariantsWithInventoryQuantityForSalesChannel(ctx context.Context, q *query.Query, variants []any, salesChannelIDs []any, keyChannelIDs []any) error {
	var channel string
	switch {
	case len(keyChannelIDs) == 1:
		channel, _ = keyChannelIDs[0].(string)
	case len(salesChannelIDs) == 1:
		channel, _ = salesChannelIDs[0].(string)
	default:
		return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Inventory availability cannot be calculated in the given context. Either provide a single sales channel id or configure a single sales channel in the publishable key")
	}
	var toWrap []*dal.Record
	var ids []any
	for _, v := range variants {
		r, ok := v.(*dal.Record)
		if !ok || r == nil {
			continue
		}
		id, _ := r.Get("id")
		if s, ok := id.(string); ok && s != "" {
			toWrap = append(toWrap, r)
			ids = append(ids, s)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	avail, err := GetVariantAvailability(ctx, q, ids, channel)
	if err != nil {
		return err
	}
	for _, v := range toWrap {
		mi, _ := v.Get("manage_inventory")
		if b, _ := mi.(bool); !b {
			continue
		}
		id, _ := v.Get("id")
		v.Set("inventory_quantity", avail[id.(string)])
	}
	return nil
}

// WrapVariantsWithTotalInventoryQuantity mirrors
// wrapVariantsWithTotalInventoryQuantity.
func WrapVariantsWithTotalInventoryQuantity(ctx context.Context, q *query.Query, variants []any) error {
	var toWrap []*dal.Record
	var ids []any
	for _, v := range variants {
		r, ok := v.(*dal.Record)
		if !ok || r == nil {
			continue
		}
		id, _ := r.Get("id")
		if s, ok := id.(string); ok && s != "" {
			toWrap = append(toWrap, r)
			ids = append(ids, s)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	avail, err := GetTotalVariantAvailability(ctx, q, ids)
	if err != nil {
		return err
	}
	for _, v := range toWrap {
		mi, _ := v.Get("manage_inventory")
		if b, _ := mi.(bool); !b {
			continue
		}
		id, _ := v.Get("id")
		v.Set("inventory_quantity", avail[id.(string)])
	}
	return nil
}
