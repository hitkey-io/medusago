// Package services mirrors @medusajs/inventory/src/services (the
// reservation item methods; the other methods use the generic service).
package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/types/src/joiner"
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
	"github.com/hitkey-io/medusago/packages/modules/inventory/src/models"
)

// InventoryModuleService mirrors InventoryModuleService.
type InventoryModuleService struct {
	*utilsmodulessdk.MedusaService
}

// NewInventoryModuleService builds the service.
func NewInventoryModuleService(opts modulessdk.ServiceOptions, joinerConfig *joiner.ModuleJoinerConfig) (*InventoryModuleService, error) {
	return &InventoryModuleService{MedusaService: utilsmodulessdk.NewMedusaService(opts.DB, models.All, joinerConfig)}, nil
}

func get(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}

func str(r *dal.Record, k string) string {
	if v := get(r, k); v != nil {
		return fmt.Sprint(v)
	}
	return ""
}

// bn returns the exact value of a big number property (its raw value when
// loaded).
func bn(r *dal.Record, k string) any {
	if raw := get(r, "raw_"+k); raw != nil {
		return raw
	}
	return get(r, k)
}

func bnString(d *totals.BN) string { return d.Text('f') }

type checkLevel struct {
	inventoryItemID string
	locationID      string
	quantity        any
	allowBackorder  bool
}

// adjustments is inventory_item_id -> location_id -> adjustment.
type adjustments map[string]map[string]*totals.BN

func (a adjustments) get(item, location string) *totals.BN {
	if m := a[item]; m != nil {
		return m[location]
	}
	return nil
}

func (a adjustments) set(item, location string, v *totals.BN) {
	if a[item] == nil {
		a[item] = map[string]*totals.BN{}
	}
	a[item][location] = v
}

func (a adjustments) current(item, location string) *totals.BN {
	if v := a.get(item, location); v != nil {
		return v
	}
	return totals.Convert(0)
}

// ensureInventoryLevels mirrors ensureInventoryLevels.
func (s *InventoryModuleService) ensureInventoryLevels(ctx context.Context, data []checkLevel, validateQuantity bool) ([]*dal.Record, error) {
	or := make([]any, 0, len(data)+1)
	idf := dal.NewRecord()
	idf.Set("id", []any{})
	or = append(or, idf)
	for _, d := range data {
		f := dal.NewRecord()
		f.Set("location_id", d.locationID)
		f.Set("inventory_item_id", d.inventoryItemID)
		or = append(or, f)
	}
	filter := dal.NewRecord()
	filter.Set("$or", or)
	levels, err := s.Internal("InventoryLevel").List(ctx, filter, utilsmodulessdk.FindConfig{})
	if err != nil {
		return nil, err
	}
	byItemLocation := map[string]map[string]*dal.Record{}
	for _, l := range levels {
		item := str(l, "inventory_item_id")
		if byItemLocation[item] == nil {
			byItemLocation[item] = map[string]*dal.Record{}
		}
		byItemLocation[item][str(l, "location_id")] = l
	}
	var missing []string
	for _, d := range data {
		if byItemLocation[d.inventoryItemID][d.locationID] == nil {
			missing = append(missing, fmt.Sprintf("Item %s is not stocked at location %s", d.inventoryItemID, d.locationID))
		}
	}
	if len(missing) > 0 {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, strings.Join(missing, ", "))
	}
	if validateQuantity {
		for _, d := range data {
			if d.allowBackorder {
				continue
			}
			level := byItemLocation[d.inventoryItemID][d.locationID]
			if totals.Lt(bn(level, "available_quantity"), d.quantity) {
				return nil, common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, fmt.Sprintf("Not enough stock available for item %s at location %s", d.inventoryItemID, d.locationID))
			}
		}
	}
	return levels, nil
}

func (s *InventoryModuleService) applyAdjustments(ctx context.Context, levels []*dal.Record, adj adjustments) error {
	var updates []any
	for _, l := range levels {
		a := adj.get(str(l, "inventory_item_id"), str(l, "location_id"))
		if a == nil || a.IsZero() {
			continue
		}
		u := dal.NewRecord()
		u.Set("id", get(l, "id"))
		u.Set("reserved_quantity", bnString(totals.Add(bn(l, "reserved_quantity"), a)))
		updates = append(updates, u)
	}
	if len(updates) == 0 {
		return nil
	}
	_, err := s.Internal("InventoryLevel").Update(ctx, updates)
	return err
}

func truthy(v any) bool {
	b, _ := v.(bool)
	return b
}

// CreateReservationItems mirrors createReservationItems.
func (s *InventoryModuleService) CreateReservationItems(ctx context.Context, input []*dal.Record) ([]*dal.Record, error) {
	var out []*dal.Record
	err := s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		s := &InventoryModuleService{MedusaService: tx}
		check := make([]checkLevel, len(input))
		for i, in := range input {
			check[i] = checkLevel{inventoryItemID: str(in, "inventory_item_id"), locationID: str(in, "location_id"), quantity: get(in, "quantity"), allowBackorder: truthy(get(in, "allow_backorder"))}
		}
		levels, err := s.ensureInventoryLevels(ctx, check, true)
		if err != nil {
			return err
		}
		created, err := s.Internal("ReservationItem").Create(ctx, input)
		if err != nil {
			return err
		}
		adj := adjustments{}
		for _, in := range input {
			item, loc := str(in, "inventory_item_id"), str(in, "location_id")
			adj.set(item, loc, totals.Add(adj.current(item, loc), get(in, "quantity")))
		}
		if err := s.applyAdjustments(ctx, levels, adj); err != nil {
			return err
		}
		out = created
		return nil
	})
	return out, err
}

// UpdateReservationItems mirrors updateReservationItems.
func (s *InventoryModuleService) UpdateReservationItems(ctx context.Context, input []*dal.Record) ([]*dal.Record, error) {
	var out []*dal.Record
	err := s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		s := &InventoryModuleService{MedusaService: tx}
		ids := make([]any, len(input))
		for i, in := range input {
			ids[i] = get(in, "id")
		}
		f := dal.NewRecord()
		f.Set("id", ids)
		reservations, err := s.Internal("ReservationItem").List(ctx, f, utilsmodulessdk.FindConfig{})
		if err != nil {
			return err
		}
		byID := map[string]*dal.Record{}
		for _, r := range reservations {
			byID[str(r, "id")] = r
		}
		var diff []string
		for _, id := range ids {
			if byID[fmt.Sprint(id)] == nil {
				diff = append(diff, fmt.Sprint(id))
			}
		}
		if len(diff) > 0 {
			return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Reservation item with id %s not found", strings.Join(diff, ", ")))
		}
		adj := adjustments{}
		for _, u := range input {
			r := byID[str(u, "id")]
			item, loc := str(r, "inventory_item_id"), str(r, "location_id")
			newLoc, hasLoc := u.Get("location_id")
			qty, hasQty := u.Get("quantity")
			if hasLoc && newLoc != nil && fmt.Sprint(newLoc) != loc {
				adj.set(item, loc, totals.Sub(adj.current(item, loc), bn(r, "quantity")))
				q := qty
				if q == nil || totals.Eq(q, 0) {
					q = bn(r, "quantity")
				}
				nl := fmt.Sprint(newLoc)
				adj.set(item, nl, totals.Add(adj.current(item, nl), q))
			} else if hasQty && qty != nil && !totals.Eq(qty, bn(r, "quantity")) {
				adj.set(item, loc, totals.Add(adj.current(item, loc), totals.Sub(qty, bn(r, "quantity"))))
			}
		}
		check := make([]checkLevel, len(input))
		for i, u := range input {
			r := byID[str(u, "id")]
			var adjustment any = 0
			if qty := get(u, "quantity"); qty != nil && !totals.Eq(qty, 0) {
				d := totals.Sub(qty, bn(r, "quantity"))
				if d.Sign() < 0 {
					adjustment = 0
				} else {
					adjustment = d
				}
			}
			loc := str(r, "location_id")
			if v := get(u, "location_id"); v != nil {
				loc = fmt.Sprint(v)
			}
			check[i] = checkLevel{inventoryItemID: str(r, "inventory_item_id"), locationID: loc, quantity: adjustment,
				allowBackorder: truthy(get(u, "allow_backorder")) || truthy(get(r, "allow_backorder"))}
		}
		levels, err := s.ensureInventoryLevels(ctx, check, true)
		if err != nil {
			return err
		}
		updates := make([]any, len(input))
		for i, u := range input {
			updates[i] = u
		}
		res, err := s.Internal("ReservationItem").Update(ctx, updates)
		if err != nil {
			return err
		}
		if err := s.applyAdjustments(ctx, levels, adj); err != nil {
			return err
		}
		switch t := res.(type) {
		case []*dal.Record:
			out = t
		case *dal.Record:
			out = []*dal.Record{t}
		}
		return nil
	})
	return out, err
}

// adjustForReservations mirrors adjustInventoryLevelsForReservations_.
func (s *InventoryModuleService) adjustForReservations(ctx context.Context, reservations []*dal.Record, isDelete bool) error {
	if len(reservations) == 0 {
		return nil
	}
	multiplier := 1
	if isDelete {
		multiplier = -1
	}
	check := make([]checkLevel, len(reservations))
	adj := adjustments{}
	for i, r := range reservations {
		item, loc := str(r, "inventory_item_id"), str(r, "location_id")
		check[i] = checkLevel{inventoryItemID: item, locationID: loc}
		adj.set(item, loc, totals.Add(adj.current(item, loc), totals.Mult(bn(r, "quantity"), multiplier)))
	}
	levels, err := s.ensureInventoryLevels(ctx, check, false)
	if err != nil {
		return err
	}
	return s.applyAdjustments(ctx, levels, adj)
}

func (s *InventoryModuleService) listReservations(ctx context.Context, ids []any, withDeleted bool) ([]*dal.Record, error) {
	f := dal.NewRecord()
	f.Set("id", ids)
	if withDeleted {
		ne := dal.NewRecord()
		ne.Set("$ne", nil)
		f.Set("deleted_at", ne)
	}
	return s.Internal("ReservationItem").List(ctx, f, utilsmodulessdk.FindConfig{WithDeleted: withDeleted})
}

// DeleteReservationItems mirrors deleteReservationItems.
func (s *InventoryModuleService) DeleteReservationItems(ctx context.Context, ids []any) error {
	return s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		s := &InventoryModuleService{MedusaService: tx}
		reservations, err := s.listReservations(ctx, ids, false)
		if err != nil {
			return err
		}
		if _, err := s.Internal("ReservationItem").Delete(ctx, ids); err != nil {
			return err
		}
		return s.adjustForReservations(ctx, reservations, true)
	})
}

// SoftDeleteReservationItems mirrors softDeleteReservationItems.
func (s *InventoryModuleService) SoftDeleteReservationItems(ctx context.Context, ids []any) error {
	return s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		s := &InventoryModuleService{MedusaService: tx}
		reservations, err := s.listReservations(ctx, ids, false)
		if err != nil {
			return err
		}
		if _, err := s.Internal("ReservationItem").SoftDelete(ctx, ids); err != nil {
			return err
		}
		return s.adjustForReservations(ctx, reservations, true)
	})
}

// RestoreReservationItems mirrors restoreReservationItems.
func (s *InventoryModuleService) RestoreReservationItems(ctx context.Context, ids []any) error {
	return s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		s := &InventoryModuleService{MedusaService: tx}
		reservations, err := s.listReservations(ctx, ids, true)
		if err != nil {
			return err
		}
		if _, err := s.Internal("ReservationItem").Restore(ctx, ids); err != nil {
			return err
		}
		return s.adjustForReservations(ctx, reservations, false)
	})
}

// RetrieveAvailableQuantity mirrors retrieveAvailableQuantity.
func (s *InventoryModuleService) RetrieveAvailableQuantity(ctx context.Context, inventoryItemID any, locationIDs []any) (*totals.BN, error) {
	if len(locationIDs) == 0 {
		return totals.Convert(0), nil
	}
	if _, err := s.Internal("InventoryItem").Retrieve(ctx, inventoryItemID, utilsmodulessdk.FindConfig{Select: []string{"id"}}); err != nil {
		return nil, err
	}
	f := dal.NewRecord()
	f.Set("location_id", locationIDs)
	f.Set("inventory_item_id", inventoryItemID)
	levels, err := s.Internal("InventoryLevel").List(ctx, f, utilsmodulessdk.FindConfig{})
	if err != nil {
		return nil, err
	}
	parts := []any{0}
	for _, l := range levels {
		st, _ := l.Get("raw_stocked_quantity")
		rs, _ := l.Get("raw_reserved_quantity")
		parts = append(parts, totals.Sub(st, rs))
	}
	return totals.Add(parts...), nil
}

// ConfirmInventory mirrors confirmInventory.
func (s *InventoryModuleService) ConfirmInventory(ctx context.Context, inventoryItemID any, locationIDs []any, quantity any) (bool, error) {
	available, err := s.RetrieveAvailableQuantity(ctx, inventoryItemID, locationIDs)
	if err != nil {
		return false, err
	}
	return totals.Gte(available, quantity), nil
}

// AdjustInventory mirrors adjustInventory (array input of
// { inventoryItemId, locationId, adjustment }).
func (s *InventoryModuleService) AdjustInventory(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	var results []*dal.Record
	for _, d := range data {
		var level *dal.Record
		err := s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
			f := dal.NewRecord()
			f.Set("inventory_item_id", get(d, "inventoryItemId"))
			f.Set("location_id", get(d, "locationId"))
			levels, err := tx.Internal("InventoryLevel").List(ctx, f, utilsmodulessdk.FindConfig{})
			if err != nil {
				return err
			}
			if len(levels) == 0 {
				return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("Inventory level for item %v and location %v not found", get(d, "inventoryItemId"), get(d, "locationId")))
			}
			u := dal.NewRecord()
			u.Set("id", get(levels[0], "id"))
			u.Set("stocked_quantity", totals.Add(get(levels[0], "stocked_quantity"), get(d, "adjustment")))
			res, err := tx.Internal("InventoryLevel").Update(ctx, u)
			if err != nil {
				return err
			}
			level, _ = res.(*dal.Record)
			return nil
		})
		if err != nil {
			return nil, err
		}
		results = append(results, level)
	}
	return results, nil
}

func (s *InventoryModuleService) reservationIDsByLineItem(ctx context.Context, lineItemIDs []any, deleted bool) ([]any, error) {
	f := dal.NewRecord()
	f.Set("line_item_id", lineItemIDs)
	if deleted {
		ne := dal.NewRecord()
		ne.Set("$ne", nil)
		f.Set("deleted_at", ne)
	}
	rows, err := s.Internal("ReservationItem").List(ctx, f, utilsmodulessdk.FindConfig{Select: []string{"id"}, WithDeleted: deleted})
	if err != nil {
		return nil, err
	}
	var ids []any
	for _, r := range rows {
		ids = append(ids, get(r, "id"))
	}
	return ids, nil
}

// DeleteReservationItemsByLineItem mirrors deleteReservationItemsByLineItem.
func (s *InventoryModuleService) DeleteReservationItemsByLineItem(ctx context.Context, lineItemIDs []any) error {
	ids, err := s.reservationIDsByLineItem(ctx, lineItemIDs, false)
	if err != nil || len(ids) == 0 {
		return err
	}
	return s.SoftDeleteReservationItems(ctx, ids)
}

// RestoreReservationItemsByLineItem mirrors restoreReservationItemsByLineItem.
func (s *InventoryModuleService) RestoreReservationItemsByLineItem(ctx context.Context, lineItemIDs []any) error {
	ids, err := s.reservationIDsByLineItem(ctx, lineItemIDs, true)
	if err != nil || len(ids) == 0 {
		return err
	}
	return s.RestoreReservationItems(ctx, ids)
}
