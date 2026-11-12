// Package services mirrors @medusajs/cart/src/services.
package services

import (
	"context"
	"slices"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/types/src/joiner"
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
	carttotals "github.com/hitkey-io/medusago/packages/core/utils/src/totals/cart"
	"github.com/hitkey-io/medusago/packages/modules/cart/src/models"
)

// CartModuleService mirrors CartModuleService.
type CartModuleService struct {
	*utilsmodulessdk.MedusaService
}

// NewCartModuleService builds the service.
func NewCartModuleService(opts modulessdk.ServiceOptions, joinerConfig *joiner.ModuleJoinerConfig) (*CartModuleService, error) {
	return &CartModuleService{MedusaService: utilsmodulessdk.NewMedusaService(opts.DB, models.All, joinerConfig)}, nil
}

func (s *CartModuleService) with(tx *utilsmodulessdk.MedusaService) *CartModuleService {
	return &CartModuleService{MedusaService: tx}
}

var totalFields = []string{"*", "total", "subtotal", "tax_total", "discount_total", "discount_tax_total", "original_total", "original_tax_total",
	"item_total", "item_subtotal", "item_tax_total", "original_item_total", "original_item_subtotal", "original_item_tax_total",
	"shipping_total", "shipping_subtotal", "shipping_tax_total", "original_shipping_tax_total", "original_shipping_subtotal", "original_shipping_total"}

var requiredFieldsForTotals = []string{"items", "credit_lines", "items.tax_lines", "items.adjustments", "shipping_methods", "shipping_methods.tax_lines", "shipping_methods.adjustments"}

// shouldIncludeTotals mirrors shouldIncludeTotals (and
// addRelationsToCalculateTotals).
func shouldIncludeTotals(config *utilsmodulessdk.FindConfig) bool {
	sel := config.Select
	if sel == nil {
		sel = []string{"*"}
	}
	include := false
	for _, f := range sel {
		if slices.Contains(totalFields, f) {
			include = true
			break
		}
	}
	if !include {
		return false
	}
	rels := append([]string{}, config.Relations...)
	for _, r := range requiredFieldsForTotals {
		if !slices.Contains(rels, r) {
			rels = append(rels, r)
		}
	}
	config.Relations = rels
	if config.Select != nil {
		var kept []string
		for _, f := range config.Select {
			skip := slices.Contains(totalFields, f)
			for _, r := range requiredFieldsForTotals {
				if strings.HasPrefix(f, r) {
					skip = true
				}
			}
			if !skip {
				kept = append(kept, f)
			}
		}
		if kept == nil {
			kept = []string{}
		}
		config.Select = kept
	}
	return true
}

func newObject() totals.Object { return dal.NewRecord() }

func decorate(cart *dal.Record) {
	carttotals.DecorateCartTotals(cart, false)
	totals.CreateRawPropertiesFromBigNumber(cart, newObject)
}

// ListBySuffix dispatches list<suffix> (listCarts computes totals).
func (s *CartModuleService) ListBySuffix(ctx context.Context, suffix string, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]*dal.Record, error) {
	if suffix == "Carts" {
		include := shouldIncludeTotals(&config)
		rows, err := s.MedusaService.ListBySuffix(ctx, suffix, filters, config)
		if err != nil {
			return nil, err
		}
		if include {
			for _, c := range rows {
				decorate(c)
			}
		}
		return rows, nil
	}
	return s.MedusaService.ListBySuffix(ctx, suffix, filters, config)
}

// ListAndCountBySuffix dispatches listAndCount<suffix>.
func (s *CartModuleService) ListAndCountBySuffix(ctx context.Context, suffix string, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]*dal.Record, int, error) {
	if suffix == "Carts" {
		include := shouldIncludeTotals(&config)
		rows, n, err := s.MedusaService.ListAndCountBySuffix(ctx, suffix, filters, config)
		if err != nil {
			return nil, 0, err
		}
		if include {
			for _, c := range rows {
				decorate(c)
			}
		}
		return rows, n, nil
	}
	return s.MedusaService.ListAndCountBySuffix(ctx, suffix, filters, config)
}

// RetrieveCart mirrors retrieveCart.
func (s *CartModuleService) RetrieveCart(ctx context.Context, id any, config utilsmodulessdk.FindConfig) (*dal.Record, error) {
	include := shouldIncludeTotals(&config)
	cart, err := s.Internal("Cart").Retrieve(ctx, id, config)
	if err != nil {
		return nil, err
	}
	if include {
		decorate(cart)
	}
	return cart, nil
}

func copyRecord(r *dal.Record) *dal.Record {
	out := dal.NewRecord()
	if r != nil {
		for _, k := range r.Keys() {
			v, _ := r.Get(k)
			out.Set(k, v)
		}
	}
	return out
}

func get(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}

// CreateCarts mirrors createCarts.
func (s *CartModuleService) CreateCarts(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	var out []*dal.Record
	err := s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		var carts []*dal.Record
		var items []*dal.Record
		var ids []any
		for _, d := range data {
			c := copyRecord(d)
			c.Delete("items")
			if code, ok := get(c, "currency_code").(string); ok && code != "" {
				c.Set("currency_code", strings.ToLower(code))
			}
			idIn, _ := get(c, "id").(string)
			id := common.GenerateEntityID(idIn, "cart")
			c.Set("id", id)
			carts = append(carts, c)
			ids = append(ids, id)
			if list, ok := get(d, "items").([]any); ok {
				for _, it := range list {
					if r, ok := it.(*dal.Record); ok {
						li := copyRecord(r)
						li.Set("cart_id", id)
						items = append(items, li)
					}
				}
			} else if list, ok := get(d, "items").([]*dal.Record); ok {
				for _, r := range list {
					li := copyRecord(r)
					li.Set("cart_id", id)
					items = append(items, li)
				}
			}
		}
		if _, err := tx.Internal("Cart").Create(ctx, carts); err != nil {
			return err
		}
		if len(items) > 0 {
			if _, err := tx.Internal("LineItem").Create(ctx, items); err != nil {
				return err
			}
		}
		f := dal.NewRecord()
		f.Set("id", ids)
		full, err := tx.Internal("Cart").List(ctx, f, utilsmodulessdk.FindConfig{Relations: []string{"shipping_address", "billing_address"}})
		if err != nil {
			return err
		}
		for _, id := range ids {
			for _, c := range full {
				if get(c, "id") == id {
					// the list returns the carts created in the same entity
					// manager: their collections are initialized empty (the
					// line items were created separately) and their shadow
					// foreign keys assigned
					for _, k := range []string{"items", "credit_lines", "shipping_methods"} {
						c.Set(k, []any{})
					}
					for _, k := range []string{"shipping_address_id", "billing_address_id"} {
						if !c.Has(k) {
							c.Set(k, nil)
						}
					}
					out = append(out, c)
				}
			}
		}
		return nil
	})
	return out, err
}

// AddLineItems mirrors addLineItems(data[]) (bulk, with cart_id).
func (s *CartModuleService) AddLineItems(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	return s.Internal("LineItem").Create(ctx, data)
}

// setTaxLines mirrors setLineItemTaxLines_ / setShippingMethodTaxLines_.
func (s *CartModuleService) setTaxLines(ctx context.Context, className, prefix, relation, cartID string, taxLines []*dal.Record) ([]*dal.Record, error) {
	var out []*dal.Record
	err := s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		var ids []any
		for _, l := range taxLines {
			idIn, _ := get(l, "id").(string)
			id := common.GenerateEntityID(idIn, prefix)
			l.Set("id", id)
			ids = append(ids, id)
		}
		del := dal.NewRecord()
		rel := dal.NewRecord()
		rel.Set("cart_id", cartID)
		del.Set(relation, rel)
		if len(ids) > 0 {
			nin := dal.NewRecord()
			nin.Set("$nin", ids)
			del.Set("id", nin)
		}
		if len(taxLines) > 0 {
			var err error
			out, err = tx.Internal(className).Upsert(ctx, taxLines)
			if err != nil {
				return err
			}
		}
		existing, err := tx.Internal(className).List(ctx, del, utilsmodulessdk.FindConfig{Select: []string{"id"}})
		if err != nil {
			return err
		}
		var delIDs []any
		for _, e := range existing {
			delIDs = append(delIDs, get(e, "id"))
		}
		if len(delIDs) > 0 {
			if _, err := tx.Internal(className).SoftDelete(ctx, delIDs); err != nil {
				return err
			}
		}
		return nil
	})
	if out == nil {
		out = []*dal.Record{}
	}
	return out, err
}

// SetLineItemTaxLines mirrors setLineItemTaxLines.
func (s *CartModuleService) SetLineItemTaxLines(ctx context.Context, cartID string, taxLines []*dal.Record) ([]*dal.Record, error) {
	return s.setTaxLines(ctx, "LineItemTaxLine", "calitxl", "item", cartID, taxLines)
}

// SetShippingMethodTaxLines mirrors setShippingMethodTaxLines.
func (s *CartModuleService) SetShippingMethodTaxLines(ctx context.Context, cartID string, taxLines []*dal.Record) ([]*dal.Record, error) {
	return s.setTaxLines(ctx, "ShippingMethodTaxLine", "casmtxl", "shipping_method", cartID, taxLines)
}
