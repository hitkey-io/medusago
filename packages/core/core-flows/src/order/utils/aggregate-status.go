// Package utils mirrors @medusajs/core-flows/src/order/utils.
package utils

import (
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/utils/src/defaults"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
)

func records(v any) []*dal.Record {
	switch t := v.(type) {
	case []*dal.Record:
		return t
	case []any:
		var out []*dal.Record
		for _, x := range t {
			if r, ok := x.(*dal.Record); ok {
				out = append(out, r)
			}
		}
		return out
	}
	return nil
}

func get(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}

func truthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case bool:
		return t
	case string:
		return t != ""
	case float64:
		return t != 0
	}
	return true
}

func decimalDigits(code string) int {
	up := strings.ToUpper(code)
	for _, c := range defaults.DefaultCurrencies {
		if c.Code == up {
			return c.DecimalDigits
		}
	}
	return -1
}

var paymentStatusOrder = []string{"not_paid", "awaiting", "captured", "partially_captured", "partially_refunded", "refunded", "canceled", "requires_action", "authorized", "partially_authorized"}

// GetLastPaymentStatus mirrors getLastPaymentStatus.
func GetLastPaymentStatus(order *dal.Record) string {
	code, _ := get(order, "currency_code").(string)
	epsilon := totals.GetEpsilonFromDecimalPrecision(decimalDigits(code))
	status := map[string]float64{}
	for _, s := range paymentStatusOrder {
		status[s] = 0
	}
	pcs := records(get(order, "payment_collections"))
	for _, pc := range pcs {
		captured := get(pc, "captured_amount")
		amount := get(pc, "amount")
		if totals.Gt(captured, 0) || (amount != nil && totals.Eq(amount, 0)) {
			if totals.Lte(totals.Sub(amount, captured), epsilon) {
				status["captured"]++
			} else {
				status["captured"] += 0.5
			}
		}
		refunded := get(pc, "refunded_amount")
		if totals.Gt(refunded, 0) {
			if totals.Lte(totals.Sub(amount, refunded), epsilon) {
				status["refunded"]++
			} else {
				status["refunded"] += 0.5
			}
		}
		s, _ := get(pc, "status").(string)
		status[s]++
	}
	total := float64(len(pcs))
	exceptCanceled := total - status["canceled"]
	switch {
	case status["requires_action"] > 0:
		return "requires_action"
	case status["refunded"] > 0:
		if status["refunded"] == status["captured"] {
			return "refunded"
		}
		return "partially_refunded"
	case status["captured"] > 0:
		if status["captured"] == exceptCanceled {
			return "captured"
		}
		return "partially_captured"
	case status["authorized"] > 0:
		if status["authorized"] == exceptCanceled {
			return "authorized"
		}
		return "partially_authorized"
	case status["canceled"] > 0 && status["canceled"] == total:
		return "canceled"
	case status["awaiting"] > 0:
		return "awaiting"
	}
	return "not_paid"
}

// GetLastFulfillmentStatus mirrors getLastFulfillmentStatus.
func GetLastFulfillmentStatus(order *dal.Record) string {
	status := map[string]float64{}
	statusMap := [][2]string{{"canceled_at", "canceled"}, {"delivered_at", "delivered"}, {"shipped_at", "shipped"}, {"packed_at", "fulfilled"}}
	fs := records(get(order, "fulfillments"))
	for _, f := range fs {
		for _, kv := range statusMap {
			if truthy(get(f, kv[0])) {
				status[kv[1]]++
				break
			}
		}
	}
	total := float64(len(fs))
	exceptCanceled := total - status["canceled"]
	hasUnfulfilled := false
	for _, item := range records(get(order, "items")) {
		detail, _ := get(item, "detail").(*dal.Record)
		rfq := get(detail, "raw_fulfilled_quantity")
		if rfq != nil && totals.Lt(rfq, get(item, "raw_quantity")) {
			hasUnfulfilled = true
			break
		}
	}
	for _, s := range []struct{ key, full, partial string }{
		{"delivered", "delivered", "partially_delivered"},
		{"shipped", "shipped", "partially_shipped"},
		{"fulfilled", "fulfilled", "partially_fulfilled"},
	} {
		if status[s.key] > 0 {
			if status[s.key] == exceptCanceled && !hasUnfulfilled {
				return s.full
			}
			return s.partial
		}
	}
	if status["canceled"] > 0 && status["canceled"] == total {
		return "canceled"
	}
	return "not_fulfilled"
}
