package utils

import (
	"fmt"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
)

// RequiredVariantFieldsForInventoryConfirmation mirrors requiredVariantFieldsForInventoryConfirmation.
var RequiredVariantFieldsForInventoryConfirmation = []string{
	"manage_inventory", "allow_backorder", "inventory_items.inventory_item_id", "inventory_items.required_quantity",
	"inventory_items.inventory.location_levels.stocked_quantity", "inventory_items.inventory.location_levels.reserved_quantity",
	"inventory_items.inventory.location_levels.raw_stocked_quantity", "inventory_items.inventory.location_levels.raw_reserved_quantity",
	"inventory_items.inventory.location_levels.location_id", "inventory_items.inventory.location_levels.stock_locations.id",
	"inventory_items.inventory.location_levels.stock_locations.name", "inventory_items.inventory.location_levels.stock_locations.sales_channels.id",
	"inventory_items.inventory.location_levels.stock_locations.sales_channels.name",
}

// DeepFlatMap mirrors deepFlatMap: the callback receives the objects
// along the path (keyed by path segment).
func DeepFlatMap(data any, path string, callback func(ctx map[string]*dal.Record)) {
	const root = "root_"
	keys := append([]string{root}, strings.Split(path, ".")...)
	lastKey := keys[len(keys)-1]
	type frame struct {
		element any
		path    []string
		ctx     map[string]*dal.Record
	}
	clone := func(m map[string]*dal.Record) map[string]*dal.Record {
		out := make(map[string]*dal.Record, len(m)+1)
		for k, v := range m {
			out[k] = v
		}
		return out
	}
	rootEl := dal.NewRecord()
	rootEl.Set(root, data)
	queue := []frame{{rootEl, keys, map[string]*dal.Record{}}}
	for len(queue) > 0 {
		f := queue[0]
		queue = queue[1:]
		current := f.path[0]
		remaining := f.path[1:]
		el, _ := f.element.(*dal.Record)
		var value any
		if el != nil {
			value, _ = el.Get(current)
		}
		if value == nil {
			callback(clone(f.ctx))
			continue
		}
		list, isList := value.([]any)
		if !isList {
			if rs, ok := value.([]*dal.Record); ok {
				isList = true
				for _, r := range rs {
					list = append(list, r)
				}
			}
		}
		if len(remaining) == 0 {
			if isList {
				for _, item := range list {
					c := clone(f.ctx)
					c[lastKey], _ = item.(*dal.Record)
					callback(c)
				}
			} else if r, ok := value.(*dal.Record); ok {
				c := clone(f.ctx)
				c[lastKey] = r
				callback(c)
			}
			continue
		}
		if isList {
			if len(list) == 0 {
				callback(clone(f.ctx))
				continue
			}
			for _, item := range list {
				c := clone(f.ctx)
				c[current], _ = item.(*dal.Record)
				queue = append(queue, frame{item, remaining, c})
			}
		} else if r, ok := value.(*dal.Record); ok {
			c := clone(f.ctx)
			c[current] = r
			queue = append(queue, frame{r, remaining, c})
		}
	}
}

// ConfirmInventoryItem mirrors ConfirmInventoryItem.
type ConfirmInventoryItem struct {
	ID               any
	InventoryItemID  any
	RequiredQuantity any
	AllowBackorder   bool
	Quantity         any
	LocationIDs      []any
}

// ConfirmVariantInventoryInput mirrors ConfirmVariantInventoryWorkflowInputDTO.
type ConfirmVariantInventoryInput struct {
	SalesChannelID string
	Variants       []*dal.Record
	Items          []*dal.Record
	ItemsToUpdate  []*dal.Record
}

// PrepareConfirmInventoryInput mirrors prepareConfirmInventoryInput.
func PrepareConfirmInventoryInput(in ConfirmVariantInventoryInput) ([]ConfirmInventoryItem, error) {
	type pvii struct {
		variantID, inventoryItemID, requiredQuantity any
	}
	var pviiKeys []string
	pviiMap := map[string]pvii{}
	var stockLocationIDs []any
	seenLocation := map[any]bool{}
	type variantInfo struct {
		id                            any
		manageInventory, allowBackord any
	}
	var variantOrder []any
	variants := map[any]variantInfo{}
	availability := map[any]map[any]*totals.BigNumber{}
	withLocation := map[any]bool{}
	hasManaged := false
	channel := in.SalesChannelID

	itemBackorder := map[any]bool{}
	for _, item := range in.Items {
		if Truthy(Get(item, "allow_backorder")) && Truthy(Get(item, "variant_id")) {
			itemBackorder[Get(item, "variant_id")] = true
		}
	}
	for _, u := range in.ItemsToUpdate {
		upd := u
		if d := Rec(u, "data"); d != nil {
			upd = d
		}
		for _, item := range in.Items {
			if Get(item, "variant_id") == Get(upd, "variant_id") {
				if q := Get(upd, "quantity"); Truthy(q) {
					item.Set("quantity", q)
				}
				break
			}
		}
	}
	data := dal.NewRecord()
	vs := make([]any, len(in.Variants))
	for i, v := range in.Variants {
		vs[i] = v
	}
	data.Set("variants", vs)
	DeepFlatMap(data, "variants.inventory_items.inventory.location_levels.stock_locations.sales_channels", func(c map[string]*dal.Record) {
		variant := c["variants"]
		if variant == nil {
			return
		}
		ii, ll, sl, sc := c["inventory_items"], c["location_levels"], c["stock_locations"], c["sales_channels"]
		if channel != "" && sc != nil && Get(sc, "id") == channel {
			withLocation[Get(variant, "id")] = true
		}
		if ll != nil && ii != nil {
			avail := totals.Sub(Coalesce(Get(ll, "raw_stocked_quantity"), Get(ll, "stocked_quantity"), 0), Coalesce(Get(ll, "raw_reserved_quantity"), Get(ll, "reserved_quantity"), 0))
			loc := Get(ll, "location_id")
			if availability[loc] == nil {
				availability[loc] = map[any]*totals.BigNumber{}
			}
			availability[loc][Get(ii, "inventory_item_id")] = totals.NewBN(avail)
		}
		if sl != nil && sc != nil && Get(sc, "id") == channel {
			if id := Get(sl, "id"); !seenLocation[id] {
				seenLocation[id] = true
				stockLocationIDs = append(stockLocationIDs, id)
			}
		}
		if ii != nil {
			key := fmt.Sprintf("%v-%v", Get(ii, "inventory_item_id"), Get(ii, "variant_id"))
			if _, ok := pviiMap[key]; !ok {
				pviiKeys = append(pviiKeys, key)
				pviiMap[key] = pvii{Get(ii, "variant_id"), Get(ii, "inventory_item_id"), Get(ii, "required_quantity")}
			}
		}
		id := Get(variant, "id")
		if _, ok := variants[id]; !ok {
			if !hasManaged && Truthy(Get(variant, "manage_inventory")) {
				hasManaged = true
			}
			variantOrder = append(variantOrder, id)
			variants[id] = variantInfo{id, Get(variant, "manage_inventory"), Get(variant, "allow_backorder")}
		}
	})
	if !hasManaged {
		return []ConfirmInventoryItem{}, nil
	}
	if channel != "" {
		for _, id := range variantOrder {
			v := variants[id]
			if Truthy(v.manageInventory) && !withLocation[id] && !Truthy(v.allowBackord) && !itemBackorder[id] {
				return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Sales channel %s is not associated with any stock location for variant %v.", channel, id))
			}
		}
	}
	if len(pviiKeys) == 0 {
		return []ConfirmInventoryItem{}, nil
	}
	out := []ConfirmInventoryItem{}
	for _, item := range in.Items {
		v, ok := variants[Get(item, "variant_id")]
		if !ok || !Truthy(v.manageInventory) {
			continue
		}
		var own []pvii
		for _, k := range pviiKeys {
			if p := pviiMap[k]; p.variantID == Get(item, "variant_id") {
				own = append(own, p)
			}
		}
		if len(own) == 0 {
			return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Variant %v does not have any inventory items associated with it.", Get(item, "variant_id")))
		}
		for _, p := range own {
			var withAvail, withLevel []any
			for _, loc := range stockLocationIDs {
				a := availability[loc]
				var have any = 0
				if a != nil && a[p.inventoryItemID] != nil {
					have = a[p.inventoryItemID]
				}
				if totals.Gte(have, totals.Mult(p.requiredQuantity, Get(item, "quantity"))) {
					withAvail = append(withAvail, loc)
				}
				if a != nil {
					if _, ok := a[p.inventoryItemID]; ok {
						withLevel = append(withLevel, loc)
					}
				}
			}
			locs := stockLocationIDs
			if len(withAvail) > 0 {
				locs = withAvail
			} else if len(withLevel) > 0 {
				locs = withLevel
			}
			if locs == nil {
				locs = []any{}
			}
			out = append(out, ConfirmInventoryItem{
				ID: Get(item, "id"), InventoryItemID: p.inventoryItemID, RequiredQuantity: p.requiredQuantity,
				AllowBackorder: Truthy(v.allowBackord) || Truthy(Get(item, "allow_backorder")),
				Quantity:       Get(item, "quantity"), LocationIDs: locs,
			})
		}
	}
	return out, nil
}
