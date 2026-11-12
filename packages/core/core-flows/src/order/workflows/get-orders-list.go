// Package workflows mirrors @medusajs/core-flows/src/order/workflows.
package workflows

import (
	"context"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/order/utils"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// GetOrdersListWorkflowID mirrors getOrdersListWorkflowId.
const GetOrdersListWorkflowID = "get-orders-list"

// GetOrdersListInput mirrors GetOrdersListWorkflowInput.
type GetOrdersListInput struct {
	Fields    []string
	Variables *dal.Record
}

func deduplicate(list []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, x := range list {
		if !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	return out
}

// GetOrdersList mirrors getOrdersListWorkflow (it only reads, so it runs
// its steps directly).
func GetOrdersList(ctx context.Context, q *query.Query, in GetOrdersListInput) (*query.GraphResult, error) {
	fields := deduplicate(append(append([]string{}, in.Fields...),
		"id", "status", "version", "items.*",
		"payment_collections.status", "payment_collections.amount", "payment_collections.captured_amount", "payment_collections.refunded_amount",
		"fulfillments.packed_at", "fulfillments.shipped_at", "fulfillments.delivered_at", "fulfillments.canceled_at",
	))
	res, err := q.RemoteQuery(ctx, query.RemoteQueryObjectFromString(query.RemoteQueryObjectConfig{EntryPoint: "orders", Variables: in.Variables, Fields: fields}))
	if err != nil {
		return nil, err
	}
	requiredPayment, requiredFulfillment, requiredItems := false, false, false
	for _, f := range in.Fields {
		if field := strings.TrimPrefix(f, "*"); field == "items" || strings.HasPrefix(field, "items.") {
			requiredItems = true
		}
		if strings.Contains(f, "payment_collections") {
			requiredPayment = true
		}
		if strings.Contains(f, "fulfillments") {
			requiredFulfillment = true
		}
	}
	for _, row := range res.Data {
		order, ok := row.(*dal.Record)
		if !ok {
			continue
		}
		order.Set("payment_status", utils.GetLastPaymentStatus(order))
		order.Set("fulfillment_status", utils.GetLastFulfillmentStatus(order))
		if !requiredPayment {
			order.Delete("payment_collections")
		}
		if !requiredFulfillment {
			order.Delete("fulfillments")
		}
		if !requiredItems {
			order.Delete("items")
		}
	}
	return res, nil
}
