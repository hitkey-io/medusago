package services

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/order/src/utils"
)

// relatedSuffixes maps list<suffix> to the related entity class.
var relatedSuffixes = map[string]string{"Returns": "Return", "OrderClaims": "OrderClaim", "OrderExchanges": "OrderExchange"}

func relatedVersionCondition() dal.RawCondition {
	return func(alias string) string {
		return fmt.Sprintf(`%s."version" = `+versionSubQuery, alias, alias)
	}
}

// findRelated mirrors the version-aware find of the Return, OrderClaim and
// OrderExchange repositories (base-repository-find with isRelatedEntity).
func (s *OrderModuleService) findRelated(ctx context.Context, className string, filters *dal.Record, config utilsmodulessdk.FindConfig, withCount bool) ([]*dal.Record, int, error) {
	where := copyRecord(filters)
	conf := utils.RepositoryConfig{
		Fields:      append([]string(nil), config.Select...),
		HasFields:   config.Select != nil,
		Populate:    append([]string{}, config.Relations...),
		HasPopulate: true,
	}
	utils.MapRepositoryToOrderModel(&conf, true)
	where.Delete("version")

	joined := !withCount && config.Take == nil && (config.Skip == nil || *config.Skip == 0)
	populate := conf.Populate
	if joined {
		if className == "OrderClaim" {
			populate = append(populate, "claim_items")
		}
		populate = append(populate, "order.shipping_methods", "order.shipping_methods.shipping_method.adjustments", "order.summary", "shipping_methods")
		if !slices.Contains(populate, "order.items") {
			populate = append([]string{"order.items"}, populate...)
		}
	} else if className == "OrderClaim" && slices.Contains(populate, "additional_items") && !slices.Contains(populate, "claim_items") {
		populate = append(populate, "claim_items")
	}
	var rest []string
	for _, p := range populate {
		if p != "order" {
			rest = append(rest, p)
		}
	}
	populate = append([]string{"order"}, rest...)

	hasRelation := func(rel string) bool {
		for _, p := range populate {
			if p == rel || strings.HasPrefix(p, rel+".") {
				return true
			}
		}
		return false
	}
	populateWhere := map[string]*dal.Record{}
	cond := func(path string) {
		r := dal.NewRecord()
		r.Set("__raw_0", relatedVersionCondition())
		populateWhere[path] = r
	}
	if hasRelation("shipping_methods") {
		cond("shipping_methods")
		cond("order.shipping_methods")
	}
	if hasRelation("items") || hasRelation("order.items") {
		cond("order.items")
	}
	if slices.Contains(populate, "order.shipping_methods.shipping_method.adjustments") {
		r := dal.NewRecord()
		r.Set("__raw_0", adjustmentVersionCondition(nil, "order_shipping", "shipping_method_id"))
		populateWhere["order.shipping_methods.shipping_method.adjustments"] = r
	}
	fc := utilsmodulessdk.FindConfig{
		Relations:        populate,
		Take:             config.Take,
		Skip:             config.Skip,
		Order:            config.Order,
		WithDeleted:      config.WithDeleted,
		PopulateWhere:    populateWhere,
		JoinedStrategy:   joined,
		CrossModuleJoins: config.CrossModuleJoins,
	}
	if conf.HasFields {
		fields := append([]string{}, conf.Fields...)
		if !joined {
			// ensureOrderItemFieldsSelection / compensateRelationFieldsSelectionFromLoadStrategy
			for _, p := range populate {
				has := false
				for _, f := range fields {
					if strings.HasPrefix(f, p+".") {
						has = true
						break
					}
				}
				if !has {
					fields = append(fields, p+".*")
				}
			}
		}
		fc.Select = fields
	}
	internal := s.Internal(className)
	if withCount {
		return internal.ListAndCount(ctx, where, fc)
	}
	rows, err := internal.List(ctx, where, fc)
	return rows, len(rows), err
}

func (s *OrderModuleService) listRelated(ctx context.Context, className string, filters *dal.Record, config utilsmodulessdk.FindConfig, withCount bool) ([]*dal.Record, int, error) {
	include := shouldIncludeTotals(&config)
	rows, n, err := s.findRelated(ctx, className, filters, config, withCount)
	if err != nil {
		return nil, 0, err
	}
	return utils.FormatOrder(rows, utils.FormatOrderOptions{Entity: className, IncludeTotals: include}), n, nil
}

// RetrieveRelated mirrors retrieveReturn / retrieveOrderClaim / retrieveOrderExchange.
func (s *OrderModuleService) RetrieveRelated(ctx context.Context, className, id string, config utilsmodulessdk.FindConfig) (*dal.Record, error) {
	f := dal.NewRecord()
	f.Set("id", id)
	rows, _, err := s.listRelated(ctx, className, f, config, false)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, notFound(className, id)
	}
	return rows[0], nil
}

// createOrderRelatedEntity mirrors createOrderRelatedEntity_.
func (s *OrderModuleService) createOrderRelatedEntity(ctx context.Context, className string, data []*dal.Record) ([]*dal.Record, error) {
	var out []*dal.Record
	err := s.InTransaction(ctx, func(m *utilsmodulessdk.MedusaService) error {
		in := make([]*dal.Record, len(data))
		var orderIDs []any
		byOrder := map[string]*dal.Record{}
		for i, d := range data {
			in[i] = copyRecord(d)
			orderIDs = append(orderIDs, get(d, "order_id"))
			byOrder[fmt.Sprint(get(d, "order_id"))] = in[i]
		}
		f := dal.NewRecord()
		f.Set("id", orderIDs)
		orders, err := m.Internal("Order").List(ctx, f, utilsmodulessdk.FindConfig{Select: []string{"id", "version"}})
		if err != nil {
			return err
		}
		if len(orders) != len(orderIDs) {
			found := map[string]bool{}
			for _, o := range orders {
				found[fmt.Sprint(get(o, "id"))] = true
			}
			var missing []string
			for _, id := range orderIDs {
				if !found[fmt.Sprint(id)] {
					missing = append(missing, fmt.Sprint(id))
				}
			}
			return invalidData("Order could not be found: " + strings.Join(missing, ", "))
		}
		for _, o := range orders {
			if r := byOrder[fmt.Sprint(get(o, "id"))]; r != nil {
				r.Set("order_version", get(o, "version"))
			}
		}
		out, err = m.Internal(className).Create(ctx, in)
		return err
	})
	return out, err
}

// CreateReturns mirrors createReturns.
func (s *OrderModuleService) CreateReturns(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	return s.createOrderRelatedEntity(ctx, "Return", data)
}

// CreateOrderClaims mirrors createOrderClaims.
func (s *OrderModuleService) CreateOrderClaims(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	return s.createOrderRelatedEntity(ctx, "OrderClaim", data)
}

// CreateOrderExchanges mirrors createOrderExchanges.
func (s *OrderModuleService) CreateOrderExchanges(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	return s.createOrderRelatedEntity(ctx, "OrderExchange", data)
}

// CreateOrderShippingMethods mirrors createOrderShippingMethods(data[]).
func (s *OrderModuleService) CreateOrderShippingMethods(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	var out []*dal.Record
	err := s.InTransaction(ctx, func(m *utilsmodulessdk.MedusaService) error {
		var orderIDs []any
		for _, d := range data {
			orderIDs = append(orderIDs, get(d, "order_id"))
		}
		f := dal.NewRecord()
		f.Set("id", orderIDs)
		orders, err := m.Internal("Order").List(ctx, f, utilsmodulessdk.FindConfig{Select: []string{"id", "version"}})
		if err != nil {
			return err
		}
		versions := map[string]any{}
		for _, o := range orders {
			versions[fmt.Sprint(get(o, "id"))] = get(o, "version")
		}
		var rows []*dal.Record
		for _, d := range data {
			r := dal.NewRecord()
			r.Set("shipping_method", copyRecord(d))
			r.Set("order_id", get(d, "order_id"))
			r.Set("return_id", get(d, "return_id"))
			r.Set("claim_id", get(d, "claim_id"))
			r.Set("exchange_id", get(d, "exchange_id"))
			v := get(d, "version")
			if v == nil {
				v = versions[fmt.Sprint(get(d, "order_id"))]
			}
			r.Set("version", v)
			rows = append(rows, r)
		}
		created, err := m.Internal("OrderShipping").Create(ctx, rows)
		if err != nil {
			return err
		}
		var ids []any
		for _, c := range created {
			if sm, ok := get(c, "shipping_method").(*dal.Record); ok {
				ids = append(ids, get(sm, "id"))
			} else {
				ids = append(ids, get(c, "shipping_method_id"))
			}
		}
		sf := dal.NewRecord()
		sf.Set("id", ids)
		methods, err := m.Internal("OrderShippingMethod").List(ctx, sf, utilsmodulessdk.FindConfig{})
		if err != nil {
			return err
		}
		byID := map[string]*dal.Record{}
		for _, sm := range methods {
			byID[fmt.Sprint(get(sm, "id"))] = sm
		}
		for _, id := range ids {
			out = append(out, byID[fmt.Sprint(id)])
		}
		return nil
	})
	return out, err
}

// UpsertOrderShippingMethodTaxLines mirrors upsertOrderShippingMethodTaxLines.
func (s *OrderModuleService) UpsertOrderShippingMethodTaxLines(ctx context.Context, taxLines []*dal.Record) ([]*dal.Record, error) {
	return s.Internal("OrderShippingMethodTaxLine").Upsert(ctx, taxLines)
}

// UpsertOrderLineItemTaxLines mirrors upsertOrderLineItemTaxLines.
func (s *OrderModuleService) UpsertOrderLineItemTaxLines(ctx context.Context, taxLines []*dal.Record) ([]*dal.Record, error) {
	return s.Internal("OrderLineItemTaxLine").Upsert(ctx, taxLines)
}

func (s *OrderModuleService) setTaxLines(ctx context.Context, orderID, relation, className string, taxLines []*dal.Record) ([]*dal.Record, error) {
	var out []*dal.Record
	err := s.InTransaction(ctx, func(m *utilsmodulessdk.MedusaService) error {
		t := s.tx(m)
		order, err := t.RetrieveOrder(ctx, orderID, utilsmodulessdk.FindConfig{Select: []string{"id"}, Relations: []string{relation + ".tax_lines"}})
		if err != nil {
			return err
		}
		keep := map[string]bool{}
		for _, tl := range taxLines {
			if id := get(tl, "id"); id != nil {
				keep[fmt.Sprint(id)] = true
			}
		}
		var del []any
		for _, it := range records(get(order, relation)) {
			for _, tl := range records(get(it, "tax_lines")) {
				if !keep[fmt.Sprint(get(tl, "id"))] {
					del = append(del, get(tl, "id"))
				}
			}
		}
		if len(del) > 0 {
			if _, err := m.Internal(className).Delete(ctx, del); err != nil {
				return err
			}
		}
		out, err = m.Internal(className).Upsert(ctx, taxLines)
		return err
	})
	return out, err
}

// SetOrderShippingMethodTaxLines mirrors setOrderShippingMethodTaxLines.
func (s *OrderModuleService) SetOrderShippingMethodTaxLines(ctx context.Context, orderID string, taxLines []*dal.Record) ([]*dal.Record, error) {
	return s.setTaxLines(ctx, orderID, "shipping_methods", "OrderShippingMethodTaxLine", taxLines)
}

// SetOrderLineItemTaxLines mirrors setOrderLineItemTaxLines.
func (s *OrderModuleService) SetOrderLineItemTaxLines(ctx context.Context, orderID string, taxLines []*dal.Record) ([]*dal.Record, error) {
	return s.setTaxLines(ctx, orderID, "items", "OrderLineItemTaxLine", taxLines)
}
