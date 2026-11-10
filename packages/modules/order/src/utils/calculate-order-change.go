package utils

import (
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
)

// ActionContext mirrors OrderReferences.
type ActionContext struct {
	Action       *dal.Record
	CurrentOrder *dal.Record
	Summary      map[string]any
	Transactions []*dal.Record
	Actions      []*dal.Record
	Options      ProcessOptions
	IsDeduction  bool
}

// ActionTypeDefinition mirrors ActionTypeDefinition.
type ActionTypeDefinition struct {
	IsDeduction bool
	// Operation returns the calculated amount (nil: undefined).
	Operation func(c ActionContext) (any, error)
	Validate  func(c ActionContext) error
}

// ProcessOptions mirrors the processing options.
type ProcessOptions struct {
	AddActionReferenceToObject bool
}

var typeDefinitions = map[string]ActionTypeDefinition{}

// RegisterActionType mirrors OrderChangeProcessing.registerActionType.
func RegisterActionType(key string, def ActionTypeDefinition) { typeDefinitions[key] = def }

// jsonValue mirrors the numeric part of JSON.parse(JSON.stringify(v)).
func jsonValue(v any) any {
	if v == nil {
		return nil
	}
	return totals.JSONNumber(v)
}

// jsonClone mirrors JSON.parse(JSON.stringify(v)).
func jsonClone(v any) any {
	if v == nil {
		return nil
	}
	b, err := dal.MarshalValue(v)
	if err != nil {
		return v
	}
	out, err := dal.DecodeOrderedJSON(b)
	if err != nil {
		return v
	}
	return out
}

func invalid(msg string) error {
	return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, msg)
}

// OrderChangeProcessing mirrors OrderChangeProcessing.
type OrderChangeProcessing struct {
	order            *dal.Record
	transactions     []*dal.Record
	actions          []*dal.Record
	options          ProcessOptions
	actionsProcessed map[string][]*dal.Record
	groupTotal       map[string]any
	summary          map[string]any
}

// NewOrderChangeProcessing mirrors the constructor.
func NewOrderChangeProcessing(order *dal.Record, transactions, actions []*dal.Record, options ProcessOptions) *OrderChangeProcessing {
	p := &OrderChangeProcessing{options: options, actionsProcessed: map[string][]*dal.Record{}, groupTotal: map[string]any{}}
	p.order, _ = jsonClone(order).(*dal.Record)
	if p.order == nil {
		p.order = dal.NewRecord()
	}
	p.transactions = records(jsonClone(toAny(transactions)))
	p.actions = records(jsonClone(toAny(actions)))
	paid := totals.Convert(0)
	refunded := totals.Convert(0)
	transactionTotal := totals.Convert(0)
	creditLineTotal := totals.Convert(0)
	for _, cl := range records(get(p.order, "credit_lines")) {
		creditLineTotal = totals.Add(creditLineTotal, get(cl, "amount"))
	}
	for _, tr := range transactions {
		amount := jsonValue(get(tr, "amount"))
		if totals.Lt(amount, 0) {
			refunded = totals.Add(refunded, totals.Abs(amount))
		} else {
			paid = totals.Add(paid, amount)
		}
		transactionTotal = totals.Add(transactionTotal, amount)
	}
	total := get(p.order, "total")
	if total == nil {
		total = 0
	}
	p.summary = map[string]any{
		"pending_difference":   0,
		"current_order_total":  total,
		"original_order_total": total,
		"transaction_total":    transactionTotal,
		"paid_total":           paid,
		"refunded_total":       refunded,
		"credit_line_total":    creditLineTotal,
		"accounting_total":     totals.Sub(total, creditLineTotal),
	}
	return p
}

func isEventActive(action *dal.Record) bool {
	status, ok := action.Get("status")
	return !ok || status == nil || status == "pending" || status == "done"
}

// ProcessActions mirrors processActions.
func (p *OrderChangeProcessing) ProcessActions() error {
	newCreditLineTotal := totals.Convert(0)
	for _, cl := range records(get(p.order, "credit_lines")) {
		if !cl.Has("id") {
			newCreditLineTotal = totals.Add(newCreditLineTotal, get(cl, "amount"))
		}
	}
	for _, action := range p.actions {
		if err := p.processAction(action); err != nil {
			return err
		}
	}
	summary := p.summary
	for _, action := range p.actions {
		if !isEventActive(action) {
			continue
		}
		def := typeDefinitions[fmt.Sprint(get(action, "action"))]
		mult := 1
		if def.IsDeduction {
			mult = -1
		}
		amount := totals.Mult(get(action, "amount"), mult)
		if changeID := get(action, "change_id"); changeID != nil {
			k := fmt.Sprint(changeID)
			prev := p.groupTotal[k]
			if prev == nil {
				prev = 0
			}
			p.groupTotal[k] = totals.Add(prev, amount)
		}
		if get(action, "action") == "CREDIT_LINE_ADD" {
			newCreditLineTotal = totals.Add(newCreditLineTotal, amount)
			summary["current_order_total"] = totals.Sub(summary["current_order_total"], amount)
		} else {
			summary["current_order_total"] = totals.Add(summary["current_order_total"], amount)
		}
	}
	summary["credit_line_total"] = newCreditLineTotal
	summary["accounting_total"] = summary["current_order_total"]
	var amounts []any
	for _, tr := range p.transactions {
		amounts = append(amounts, get(tr, "amount"))
	}
	summary["transaction_total"] = totals.Sum(amounts...)
	summary["pending_difference"] = totals.Sub(summary["current_order_total"], summary["transaction_total"])
	return nil
}

func (p *OrderChangeProcessing) processAction(action *dal.Record) error {
	name := fmt.Sprint(get(action, "action"))
	def, ok := typeDefinitions[name]
	if !ok {
		return fmt.Errorf("Action type %s is not defined", name)
	}
	p.actionsProcessed[name] = append(p.actionsProcessed[name], action)
	c := ActionContext{Action: action, CurrentOrder: p.order, Summary: p.summary, Transactions: p.transactions, Actions: p.actions, Options: p.options, IsDeduction: def.IsDeduction}
	if def.Validate != nil {
		if err := def.Validate(c); err != nil {
			return err
		}
	}
	if def.Operation != nil {
		calculated, err := def.Operation(c)
		if err != nil {
			return err
		}
		if a, ok := action.Get("amount"); !ok || a == nil {
			if calculated == nil {
				calculated = 0
			}
			action.Set("amount", calculated)
		}
	}
	return nil
}

var summaryKeys = []string{"transaction_total", "original_order_total", "current_order_total", "pending_difference", "paid_total", "refunded_total", "credit_line_total", "accounting_total"}

func bn(v any) *totals.BigNumber {
	if v == nil {
		v = 0
	}
	return totals.NewBN(v)
}

// GetSummary mirrors getSummary (used for order creation).
func (p *OrderChangeProcessing) GetSummary() *dal.Record {
	out := dal.NewRecord()
	for _, k := range summaryKeys {
		out.Set(k, bn(p.summary[k]))
	}
	return out
}

// GetSummaryFromOrder mirrors getSummaryFromOrder.
func (p *OrderChangeProcessing) GetSummaryFromOrder(order *dal.Record) *dal.Record {
	s := p.summary
	total := get(order, "total")
	out := dal.NewRecord()
	out.Set("transaction_total", bn(s["transaction_total"]))
	out.Set("original_order_total", bn(s["original_order_total"]))
	out.Set("current_order_total", bn(total))
	out.Set("pending_difference", bn(totals.Sub(total, s["transaction_total"])))
	out.Set("paid_total", bn(s["paid_total"]))
	out.Set("refunded_total", bn(s["refunded_total"]))
	out.Set("credit_line_total", bn(s["credit_line_total"]))
	cur, _ := out.Get("current_order_total")
	out.Set("accounting_total", cur)
	return out
}

// CurrentOrder mirrors getCurrentOrder.
func (p *OrderChangeProcessing) CurrentOrder() *dal.Record { return p.order }

// CalculatedOrderChange mirrors the result of calculateOrderChange.
type CalculatedOrderChange struct {
	Instance *OrderChangeProcessing
	Summary  *dal.Record
	Order    *dal.Record
}

// CalculateOrderChange mirrors calculateOrderChange.
func CalculateOrderChange(order *dal.Record, transactions, actions []*dal.Record, options ProcessOptions) (*CalculatedOrderChange, error) {
	p := NewOrderChangeProcessing(order, transactions, actions, options)
	if err := p.ProcessActions(); err != nil {
		return nil, err
	}
	return &CalculatedOrderChange{Instance: p, Summary: p.GetSummary(), Order: p.CurrentOrder()}, nil
}
