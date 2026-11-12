// Package workflows mirrors @medusajs/core-flows/src/order/workflows.
package workflows

import (
	"context"
	"strings"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/order/utils"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// GetOrdersListWorkflowID mirrors getOrdersListWorkflowId.
const GetOrdersListWorkflowID = "get-orders-list"

// GetOrdersListInput mirrors GetOrdersListWorkflowInput.
type GetOrdersListInput struct {
	Fields    []string    `json:"fields"`
	Variables *dal.Record `json:"variables"`
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

// GetOrdersListWorkflow mirrors getOrdersListWorkflow: the orders (rows, or
// { rows, metadata } when paginated) with their payment and fulfillment
// statuses; relations needed for the statuses are dropped unless requested.
var GetOrdersListWorkflow = composer.CreateWorkflow(GetOrdersListWorkflowID,
	func(e *composer.Execution, in GetOrdersListInput) (*query.GraphResult, error) {
		fields := deduplicate(append(append([]string{}, in.Fields...),
			"id", "status", "version", "items.*",
			"payment_collections.status", "payment_collections.amount", "payment_collections.captured_amount", "payment_collections.refunded_amount",
			"fulfillments.packed_at", "fulfillments.shipped_at", "fulfillments.delivered_at", "fulfillments.canceled_at",
		))
		res, err := commonsteps.UseRemoteQueryStep.Run(e, commonsteps.RemoteStepInput{EntryPoint: "orders", Fields: fields, Variables: in.Variables})
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
	})

// GetOrdersList runs getOrdersListWorkflow (the routes run it like upstream).
func GetOrdersList(ctx context.Context, c composer.Container, in GetOrdersListInput) (*query.GraphResult, error) {
	return GetOrdersListWorkflow.Run(ctx, c, in)
}
