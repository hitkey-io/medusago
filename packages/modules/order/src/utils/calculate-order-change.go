package utils

import (
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
)

// jsonValue mirrors JSON.parse(JSON.stringify(v)) for the numeric fields
// the calculation reads: BigNumbers become numbers.
func jsonValue(v any) any {
	if v == nil {
		return nil
	}
	return totals.JSONNumber(v)
}

// OrderChangeProcessing mirrors OrderChangeProcessing (the summary part;
// change actions are not supported yet).
type OrderChangeProcessing struct {
	order        *dal.Record
	transactions []*dal.Record
	actions      []*dal.Record
	summary      map[string]any
}

// NewOrderChangeProcessing mirrors the constructor.
func NewOrderChangeProcessing(order *dal.Record, transactions, actions []*dal.Record) *OrderChangeProcessing {
	p := &OrderChangeProcessing{order: order, transactions: transactions, actions: actions}
	paid := totals.Convert(0)
	refunded := totals.Convert(0)
	transactionTotal := totals.Convert(0)
	creditLineTotal := totals.Convert(0)
	for _, cl := range records(get(order, "credit_lines")) {
		creditLineTotal = totals.Add(creditLineTotal, jsonValue(get(cl, "amount")))
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
	total := jsonValue(get(order, "total"))
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

// ProcessActions mirrors processActions.
func (p *OrderChangeProcessing) ProcessActions() error {
	newCreditLineTotal := totals.Convert(0)
	for _, cl := range records(get(p.order, "credit_lines")) {
		if !cl.Has("id") {
			newCreditLineTotal = totals.Add(newCreditLineTotal, jsonValue(get(cl, "amount")))
		}
	}
	if len(p.actions) > 0 {
		return fmt.Errorf("order change actions are not supported yet")
	}
	p.summary["credit_line_total"] = newCreditLineTotal
	p.summary["accounting_total"] = p.summary["current_order_total"]
	var amounts []any
	for _, tr := range p.transactions {
		amounts = append(amounts, jsonValue(get(tr, "amount")))
	}
	p.summary["transaction_total"] = totals.Sum(amounts...)
	p.summary["pending_difference"] = totals.Sub(p.summary["current_order_total"], p.summary["transaction_total"])
	return nil
}

func bn(v any) *totals.BigNumber {
	if d, ok := v.(*totals.BN); ok {
		return totals.NewBN(d)
	}
	return totals.NewBN(v)
}

// GetSummary mirrors getSummary (used for order creation).
func (p *OrderChangeProcessing) GetSummary() *dal.Record {
	s := p.summary
	out := dal.NewRecord()
	for _, k := range []string{"transaction_total", "original_order_total", "current_order_total", "pending_difference", "paid_total", "refunded_total", "credit_line_total", "accounting_total"} {
		out.Set(k, bn(s[k]))
	}
	return out
}

// CalculateOrderChange mirrors calculateOrderChange; it returns the summary.
func CalculateOrderChange(order *dal.Record, transactions, actions []*dal.Record) (*dal.Record, error) {
	p := NewOrderChangeProcessing(order, transactions, actions)
	if err := p.ProcessActions(); err != nil {
		return nil, err
	}
	return p.GetSummary(), nil
}
