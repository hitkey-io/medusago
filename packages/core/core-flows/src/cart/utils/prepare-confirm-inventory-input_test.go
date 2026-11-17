package utils

import (
	"strings"
	"testing"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

func decodeRecords(t *testing.T, src string) []*dal.Record {
	t.Helper()
	v, err := dal.DecodeOrderedJSON([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	var out []*dal.Record
	for _, x := range v.([]any) {
		out = append(out, x.(*dal.Record))
	}
	return out
}

func variantJSON(id, item, location, channels string) string {
	return `{"id": "` + id + `", "manage_inventory": true, "inventory_items": [{"inventory_item_id": "` + item + `", "variant_id": "` + id +
		`", "required_quantity": 1, "inventory": [{"location_levels": {"location_id": "` + location +
		`", "stocked_quantity": 10, "reserved_quantity": 0, "stock_locations": [{"id": "` + location + `", "sales_channels": ` + channels + `}]}}]}]}`
}

func TestPrepareConfirmInventoryInputErrors(t *testing.T) {
	cases := []struct {
		name, variants, items, want string
	}{
		{
			// the only stock location is linked to no sales channel
			name:     "location without sales channels",
			variants: `[` + variantJSON("pv_1", "ii_1", "sl_1", `[]`) + `]`,
			items:    `[{"variant_id": "pv_1", "quantity": 4, "id": "item_1"}]`,
			want:     "Sales channel sc_1 is not associated with any stock location for variant pv_1.",
		},
		{
			// another variant is confirmable: the variant is not skipped
			name:     "location without sales channels next to a confirmable variant",
			variants: `[` + variantJSON("pv_1", "ii_1", "sl_1", `[{"id": "sc_1"}]`) + `, ` + variantJSON("pv_2", "ii_2", "sl_2", `[]`) + `]`,
			items:    `[{"variant_id": "pv_1", "quantity": 1, "id": "item_1"}, {"variant_id": "pv_2", "quantity": 4, "id": "item_2"}]`,
			want:     "Sales channel sc_1 is not associated with any stock location for variant pv_2.",
		},
		{
			name:     "variant without inventory items",
			variants: `[{"id": "pv_1", "manage_inventory": true, "inventory_items": []}]`,
			items:    `[{"variant_id": "pv_1", "quantity": 1, "id": "item_1"}]`,
			want:     "Variant pv_1 does not have any inventory items associated with it.",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := PrepareConfirmInventoryInput(ConfirmVariantInventoryInput{
				SalesChannelID: "sc_1",
				Variants:       decodeRecords(t, c.variants),
				Items:          decodeRecords(t, c.items),
			})
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("got %v, want %q", err, c.want)
			}
		})
	}
}
