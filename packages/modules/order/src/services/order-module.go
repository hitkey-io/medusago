// Package services mirrors @medusajs/order/src/services.
package services

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/types/src/joiner"
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
	carttotals "github.com/hitkey-io/medusago/packages/core/utils/src/totals/cart"
	"github.com/hitkey-io/medusago/packages/modules/order/src/models"
	"github.com/hitkey-io/medusago/packages/modules/order/src/utils"
)

// OrderModuleService mirrors OrderModuleService.
type OrderModuleService struct {
	*utilsmodulessdk.MedusaService
}

// NewOrderModuleService builds the service.
func NewOrderModuleService(opts modulessdk.ServiceOptions, joinerConfig *joiner.ModuleJoinerConfig) (*OrderModuleService, error) {
	return &OrderModuleService{MedusaService: utilsmodulessdk.NewMedusaService(opts.DB, models.All, joinerConfig)}, nil
}

var totalFields = []string{"*", "total", "subtotal", "tax_total", "discount_total", "discount_tax_total", "original_total", "original_tax_total",
	"pending_difference", "item_total", "item_subtotal", "item_tax_total", "original_item_total", "original_item_subtotal", "original_item_tax_total",
	"shipping_total", "shipping_subtotal", "shipping_tax_total", "original_shipping_tax_total", "original_shipping_subtotal", "original_shipping_total",
	"original_total", "original_subtotal", "original_tax_total", "credit_line_total", "credit_line_tax_total", "credit_line_subtotal", "refundable_amount"}

var requiredRelationsForTotals = []string{"credit_lines", "items", "items.tax_lines", "items.adjustments", "shipping_methods", "shipping_methods.tax_lines", "shipping_methods.adjustments"}

func dedupe(list []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, x := range list {
		if !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	return out
}

// shouldIncludeTotals mirrors shouldIncludeTotals + addRelationsToCalculateTotals.
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
	config.Relations = dedupe(append(append([]string{}, config.Relations...), requiredRelationsForTotals...))
	if config.Select != nil {
		kept := []string{}
		for _, f := range config.Select {
			skip := slices.Contains(totalFields, f)
			for _, r := range requiredRelationsForTotals {
				if strings.HasPrefix(f, r) {
					skip = true
				}
			}
			if !skip {
				kept = append(kept, f)
			}
		}
		config.Select = kept
	}
	return true
}

func get(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}

func copyRecord(r *dal.Record) *dal.Record {
	out := dal.NewRecord()
	if r == nil {
		return out
	}
	for _, k := range r.Keys() {
		v, _ := r.Get(k)
		out.Set(k, v)
	}
	return out
}

func records(v any) []*dal.Record {
	switch list := v.(type) {
	case []*dal.Record:
		return list
	case []any:
		out := make([]*dal.Record, 0, len(list))
		for _, x := range list {
			if r, ok := x.(*dal.Record); ok {
				out = append(out, r)
			}
		}
		return out
	}
	return nil
}

func toAny(list []*dal.Record) []any {
	out := make([]any, len(list))
	for i, r := range list {
		out[i] = r
	}
	return out
}

const versionSubQuery = `(select "_sub0"."version" from "order" as "_sub0" where "_sub0"."id" = %s."order_id")`

// versionCondition is the populateWhere version filter of a versioned
// relation (base-repository-find configurePopulateWhere).
func versionCondition(version any) dal.RawCondition {
	return func(alias string) string {
		if version != nil {
			return fmt.Sprintf(`%s."version" = %s`, alias, sqlLiteral(version))
		}
		return fmt.Sprintf(`%s."version" = `+versionSubQuery, alias, alias)
	}
}

// adjustmentVersionCondition restricts the adjustments of the loaded items
// (or shipping methods) to the version they were loaded with, like
// loadItemAdjustments / loadShippingAdjustments.
func adjustmentVersionCondition(version any, pivotTable, fk string) dal.RawCondition {
	return func(alias string) string {
		if version != nil {
			return fmt.Sprintf(`%s."version" = %s`, alias, sqlLiteral(version))
		}
		return fmt.Sprintf(`%s."version" = (select "_p"."version" from %q as "_p" inner join "order" as "_o" on "_o"."id" = "_p"."order_id" and "_o"."version" = "_p"."version" where "_p".%q = %s.%q and "_p"."deleted_at" is null limit 1)`,
			alias, pivotTable, fk, alias, fk)
	}
}

func sqlLiteral(v any) string {
	switch t := v.(type) {
	case float64, int, int64:
		return fmt.Sprint(t)
	case string:
		return "'" + strings.ReplaceAll(t, "'", "''") + "'"
	}
	return "'" + strings.ReplaceAll(fmt.Sprint(v), "'", "''") + "'"
}

// findOrders mirrors the version-aware OrderRepository.find/findAndCount
// (utils/base-repository-find.ts) for the Order entity.
func (s *OrderModuleService) findOrders(ctx context.Context, filters *dal.Record, config utilsmodulessdk.FindConfig, withCount bool) ([]*dal.Record, int, error) {
	where := copyRecord(filters)
	conf := utils.RepositoryConfig{
		Fields:      append([]string(nil), config.Select...),
		HasFields:   config.Select != nil,
		Populate:    append([]string{}, config.Relations...),
		HasPopulate: true,
		Where:       where,
	}
	utils.MapRepositoryToOrderModel(&conf, false)

	version := get(where, "version")
	where.Delete("version")

	hasRelation := func(rel string) bool {
		for _, p := range conf.Populate {
			if p == rel || strings.HasPrefix(p, rel+".") {
				return true
			}
		}
		return false
	}
	populateWhere := map[string]*dal.Record{}
	cond := func(path string, c dal.RawCondition) {
		r := populateWhere[path]
		if r == nil {
			r = dal.NewRecord()
			populateWhere[path] = r
		}
		r.Set(fmt.Sprintf("__raw_%d", r.Len()), c)
	}
	for _, rel := range []string{"summary", "credit_lines", "items", "shipping_methods"} {
		if hasRelation(rel) || (rel == "items" && hasRelation("order.items")) {
			cond(rel, versionCondition(version))
		}
	}

	ensure := func(list []string, values ...string) []string {
		for _, v := range values {
			if !slices.Contains(list, v) {
				list = append(list, v)
			}
		}
		return list
	}
	if slices.Contains(conf.Populate, "items.item.adjustments") {
		conf.Populate = ensure(conf.Populate, "items", "items.item")
		if conf.HasFields && slices.ContainsFunc(conf.Fields, func(f string) bool { return strings.Contains(f, "items.item.") }) {
			conf.Fields = append(conf.Fields, "items.version")
		}
		cond("items.item.adjustments", adjustmentVersionCondition(version, "order_item", "item_id"))
	}
	if slices.Contains(conf.Populate, "shipping_methods.shipping_method.adjustments") {
		conf.Populate = ensure(conf.Populate, "shipping_methods", "shipping_methods.shipping_method")
		if conf.HasFields && slices.ContainsFunc(conf.Fields, func(f string) bool { return strings.Contains(f, "shipping_methods.shipping_method.") }) {
			conf.Fields = append(conf.Fields, "shipping_methods.version")
		}
		cond("shipping_methods.shipping_method.adjustments", adjustmentVersionCondition(version, "order_shipping", "shipping_method_id"))
	}

	fc := utilsmodulessdk.FindConfig{
		Relations:     conf.Populate,
		Take:          config.Take,
		Skip:          config.Skip,
		Order:         config.Order,
		WithDeleted:   config.WithDeleted,
		PopulateWhere: populateWhere,
	}
	if conf.HasFields {
		fc.Select = conf.Fields
	}
	internal := s.Internal("Order")
	if withCount {
		return internal.ListAndCount(ctx, where, fc)
	}
	rows, err := internal.List(ctx, where, fc)
	return rows, len(rows), err
}

func (s *OrderModuleService) listOrders(ctx context.Context, filters *dal.Record, config utilsmodulessdk.FindConfig, withCount bool) ([]*dal.Record, int, error) {
	include := shouldIncludeTotals(&config)
	rows, n, err := s.findOrders(ctx, filters, config, withCount)
	if err != nil {
		return nil, 0, err
	}
	return utils.FormatOrder(rows, utils.FormatOrderOptions{Entity: "Order", IncludeTotals: include}), n, nil
}

// ListBySuffix dispatches list<suffix> (listOrders formats the orders).
func (s *OrderModuleService) ListBySuffix(ctx context.Context, suffix string, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]*dal.Record, error) {
	if suffix == "Orders" {
		rows, _, err := s.listOrders(ctx, filters, config, false)
		return rows, err
	}
	return s.MedusaService.ListBySuffix(ctx, suffix, filters, config)
}

// ListAndCountBySuffix dispatches listAndCount<suffix>.
func (s *OrderModuleService) ListAndCountBySuffix(ctx context.Context, suffix string, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]*dal.Record, int, error) {
	if suffix == "Orders" {
		return s.listOrders(ctx, filters, config, true)
	}
	return s.MedusaService.ListAndCountBySuffix(ctx, suffix, filters, config)
}

// ListOrders mirrors listOrders.
func (s *OrderModuleService) ListOrders(ctx context.Context, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]*dal.Record, error) {
	rows, _, err := s.listOrders(ctx, filters, config, false)
	return rows, err
}

// RetrieveOrder mirrors retrieveOrder (with the active order change).
func (s *OrderModuleService) RetrieveOrder(ctx context.Context, id string, config utilsmodulessdk.FindConfig) (*dal.Record, error) {
	include := shouldIncludeTotals(&config)
	f := dal.NewRecord()
	f.Set("id", id)
	rows, _, err := s.findOrders(ctx, f, config, false)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, notFound("Order", id)
	}
	order := rows[0]
	change, err := s.getActiveOrderChange(ctx, id, false)
	if err != nil {
		return nil, err
	}
	if change != nil {
		order.Set("order_change", change)
	}
	return utils.FormatOrder([]*dal.Record{order}, utils.FormatOrderOptions{Entity: "Order", IncludeTotals: include})[0], nil
}

func (s *OrderModuleService) getActiveOrderChange(ctx context.Context, orderID string, includeActions bool) (*dal.Record, error) {
	sel := []string{"id", "change_type", "order_id", "return_id", "claim_id", "exchange_id", "version", "requested_at", "requested_by",
		"status", "description", "internal_note", "carry_over_promotions", "no_notification"}
	var rels []string
	var order []dal.OrderBy
	if includeActions {
		sel = append(sel, "actions")
		rels = append(rels, "actions")
		order = append(order, dal.OrderBy{Field: "actions.ordering", Direction: "ASC"})
	}
	f := dal.NewRecord()
	f.Set("order_id", orderID)
	f.Set("status", []any{"pending", "requested"})
	rows, err := s.MedusaService.ListBySuffix(ctx, "OrderChanges", f, utilsmodulessdk.FindConfig{Select: sel, Relations: rels, Order: order})
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	return rows[0], nil
}

// CreateOrders mirrors createOrders.
func (s *OrderModuleService) CreateOrders(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	var ids []any
	err := s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		created, err := (&OrderModuleService{MedusaService: tx}).createOrders(ctx, data)
		if err != nil {
			return err
		}
		for _, o := range created {
			ids = append(ids, get(o, "id"))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	f := dal.NewRecord()
	f.Set("id", ids)
	return s.ListOrders(dal.WithSharedIdentityMap(ctx), f, utilsmodulessdk.FindConfig{Relations: []string{
		"shipping_address", "billing_address", "summary", "items", "credit_lines", "items.tax_lines", "items.adjustments",
		"shipping_methods", "shipping_methods.tax_lines", "shipping_methods.adjustments", "transactions",
	}})
}

func (s *OrderModuleService) createOrderAddresses(ctx context.Context, inputs []*dal.Record) error {
	type addr struct {
		data   *dal.Record
		typ    string
		source *dal.Record
	}
	var all []addr
	for _, in := range inputs {
		if b, ok := get(in, "billing_address").(*dal.Record); ok && b != nil {
			all = append(all, addr{b, "billing", in})
		}
		if sa, ok := get(in, "shipping_address").(*dal.Record); ok && sa != nil {
			all = append(all, addr{sa, "shipping", in})
		}
	}
	if len(all) == 0 {
		return nil
	}
	data := make([]*dal.Record, len(all))
	for i, a := range all {
		data[i] = a.data
	}
	created, err := s.Internal("OrderAddress").Create(ctx, data)
	if err != nil {
		return err
	}
	for i, c := range created {
		if all[i].typ == "billing" {
			all[i].source.Set("billing_address_id", get(c, "id"))
		} else {
			all[i].source.Set("shipping_address_id", get(c, "id"))
		}
	}
	return nil
}

func (s *OrderModuleService) createOrders(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	inputs := make([]*dal.Record, len(data))
	for i, d := range data {
		inputs[i] = copyRecord(d)
	}
	if err := s.createOrderAddresses(ctx, inputs); err != nil {
		return nil, err
	}
	var lineItemsToCreate, creditLinesToCreate, createdOrders []*dal.Record
	for _, in := range inputs {
		items := records(get(in, "items"))
		shippingMethods := records(get(in, "shipping_methods"))
		creditLines := records(get(in, "credit_lines"))
		ord := copyRecord(in)
		for _, k := range []string{"items", "shipping_methods", "credit_lines", "shipping_address", "billing_address"} {
			ord.Delete(k)
		}
		if shippingMethods != nil {
			wrapped := make([]any, len(shippingMethods))
			for i, sm := range shippingMethods {
				w := dal.NewRecord()
				w.Set("shipping_method", copyRecord(sm))
				wrapped[i] = w
			}
			ord.Set("shipping_methods", wrapped)
		}

		withTotals := copyRecord(ord)
		withTotals.Set("shipping_methods", optionalList(in, "shipping_methods", shippingMethods))
		withTotals.Set("items", optionalList(in, "items", items))
		withTotals.Set("credit_lines", optionalList(in, "credit_lines", creditLines))
		carttotals.DecorateCartTotals(withTotals, false)
		calc, err := utils.CalculateOrderChange(withTotals, records(get(in, "transactions")), nil, utils.ProcessOptions{})
		if err != nil {
			return nil, err
		}
		summary := calc.Summary
		totals.CreateRawPropertiesFromBigNumber(summary, func() totals.Object { return dal.NewRecord() })
		sumRec := dal.NewRecord()
		sumRec.Set("totals", summary)
		ord.Set("summary", []any{sumRec})
		if code, ok := get(ord, "currency_code").(string); ok && code != "" {
			ord.Set("currency_code", strings.ToLower(code))
		}

		created, err := s.Internal("Order").Create(ctx, []*dal.Record{ord})
		if err != nil {
			return nil, err
		}
		order := created[0]
		for _, cl := range creditLines {
			c := dal.NewRecord()
			c.Set("amount", get(cl, "amount"))
			c.Set("reference", get(cl, "reference"))
			c.Set("reference_id", get(cl, "reference_id"))
			c.Set("metadata", get(cl, "metadata"))
			c.Set("order_id", get(order, "id"))
			creditLinesToCreate = append(creditLinesToCreate, c)
		}
		createdOrders = append(createdOrders, order)
		for _, item := range items {
			li := copyRecord(item)
			li.Set("order_id", get(order, "id"))
			lineItemsToCreate = append(lineItemsToCreate, li)
		}
	}
	if len(lineItemsToCreate) > 0 {
		if _, err := s.createOrderLineItemsBulk(ctx, lineItemsToCreate); err != nil {
			return nil, err
		}
	}
	if len(creditLinesToCreate) > 0 {
		if _, err := s.Internal("OrderCreditLine").Create(ctx, creditLinesToCreate); err != nil {
			return nil, err
		}
	}
	return createdOrders, nil
}

// optionalList keeps an absent key undefined (decorateCartTotals reads
// `items ?? []`).
func optionalList(in *dal.Record, key string, list []*dal.Record) any {
	if _, ok := in.Get(key); !ok {
		return nil
	}
	return toAny(list)
}

func (s *OrderModuleService) createOrderLineItemsBulk(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	lineItems, err := s.Internal("OrderLineItem").Create(ctx, data)
	if err != nil {
		return nil, err
	}
	var orderItems []*dal.Record
	for i, item := range lineItems {
		in := data[i]
		orderID := get(in, "order_id")
		if orderID == nil {
			continue
		}
		oi := dal.NewRecord()
		oi.Set("order_id", orderID)
		version := get(in, "version")
		if version == nil {
			version = 1
		}
		oi.Set("version", version)
		oi.Set("item_id", get(item, "id"))
		oi.Set("quantity", get(in, "quantity"))
		if m, ok := in.Get("metadata"); ok {
			oi.Set("metadata", m)
		}
		orderItems = append(orderItems, oi)
	}
	if len(orderItems) > 0 {
		if _, err := s.Internal("OrderItem").Create(ctx, orderItems); err != nil {
			return nil, err
		}
	}
	return lineItems, nil
}

// AddOrderTransactions mirrors addOrderTransactions (array input).
func (s *OrderModuleService) AddOrderTransactions(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	var out []*dal.Record
	err := s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		var ids []any
		for _, d := range data {
			ids = append(ids, get(d, "order_id"))
		}
		f := dal.NewRecord()
		f.Set("id", ids)
		orders, err := tx.Internal("Order").List(ctx, f, utilsmodulessdk.FindConfig{Select: []string{"id", "version"}})
		if err != nil {
			return err
		}
		normalized := make([]*dal.Record, len(data))
		for i, d := range data {
			n := copyRecord(d)
			for _, o := range orders {
				if get(o, "id") == get(d, "order_id") {
					n.Set("version", get(o, "version"))
				}
			}
			code, _ := get(d, "currency_code").(string)
			n.Set("currency_code", strings.ToLower(code))
			normalized[i] = n
		}
		created, err := tx.Internal("OrderTransaction").Create(ctx, normalized)
		if err != nil {
			return err
		}
		if err := (&OrderModuleService{MedusaService: tx}).updateOrderPaidRefundableAmount(ctx, created, false); err != nil {
			return err
		}
		out = created
		return nil
	})
	return out, err
}

// updateOrderPaidRefundableAmount mirrors updateOrderPaidRefundableAmount_.
func (s *OrderModuleService) updateOrderPaidRefundableAmount(ctx context.Context, transactions []*dal.Record, isRemoved bool) error {
	if len(transactions) == 0 {
		return nil
	}
	var orderIDs []any
	for _, t := range transactions {
		orderIDs = append(orderIDs, get(t, "order_id"))
	}
	f := dal.NewRecord()
	f.Set("order_id", orderIDs)
	f.Set("version", get(transactions[0], "version"))
	summaries, err := s.Internal("OrderSummary").List(ctx, f, utilsmodulessdk.FindConfig{})
	if err != nil {
		return err
	}
	op := totals.Add
	if isRemoved {
		op = totals.Sub
	}
	var updates []any
	for _, summary := range summaries {
		var trxs []*dal.Record
		for _, t := range transactions {
			if get(t, "order_id") == get(summary, "order_id") {
				trxs = append(trxs, t)
			}
		}
		if len(trxs) == 0 {
			continue
		}
		t, _ := get(summary, "totals").(*dal.Record)
		if t == nil {
			t = dal.NewRecord()
		}
		initialTrxTotal := get(t, "transaction_total")
		for _, trx := range trxs {
			amount := get(trx, "amount")
			if totals.Gt(amount, 0) {
				t.Set("paid_total", totals.NewBN(op(get(t, "paid_total"), amount)))
			} else {
				t.Set("refunded_total", totals.NewBN(op(get(t, "refunded_total"), totals.Abs(amount))))
			}
			t.Set("transaction_total", totals.NewBN(op(get(t, "transaction_total"), amount)))
		}
		initialDiff := totals.Sub(get(t, "transaction_total"), initialTrxTotal)
		t.Set("pending_difference", totals.NewBN(totals.Sub(get(t, "pending_difference"), initialDiff)))
		totals.CreateRawPropertiesFromBigNumber(summary, func() totals.Object { return dal.NewRecord() })
		u := dal.NewRecord()
		u.Set("id", get(summary, "id"))
		u.Set("totals", t)
		updates = append(updates, u)
	}
	if len(updates) == 0 {
		return nil
	}
	_, err = s.Internal("OrderSummary").Update(ctx, updates)
	return err
}

func notFound(entity, id string) error {
	return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("%s with id: %s was not found", entity, id))
}
