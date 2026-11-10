package workflows

import (
	"context"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/order/utils"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// GetOrderDetailWorkflowID mirrors getOrderDetailWorkflowId.
const GetOrderDetailWorkflowID = "get-order-detail"

// GetOrderDetailInput mirrors GetOrderDetailWorkflowInput.
type GetOrderDetailInput struct {
	Filters *dal.Record
	Fields  []string
	OrderID string
	Version any
}

// GetOrderDetail mirrors getOrderDetailWorkflow (it only reads, so it runs
// its step directly).
func GetOrderDetail(ctx context.Context, q *query.Query, in GetOrderDetailInput) (*dal.Record, error) {
	fields := deduplicate(append(append([]string{}, in.Fields...), "id", "status", "version", "payment_collections.*", "fulfillments.*"))
	variables := dal.NewRecord()
	if in.Filters != nil {
		for _, k := range in.Filters.Keys() {
			v, _ := in.Filters.Get(k)
			variables.Set(k, v)
		}
	}
	variables.Set("id", in.OrderID)
	if in.Version != nil {
		variables.Set("version", in.Version)
	}
	res, err := q.Graph(ctx, query.GraphInput{Entity: "order", Fields: fields, Filters: variables}, joiner.Options{ThrowIfKeyNotFound: true})
	if err != nil {
		return nil, err
	}
	if len(res.Data) == 0 {
		return nil, nil
	}
	order, _ := res.Data[0].(*dal.Record)
	if order == nil {
		return nil, nil
	}
	order.Set("payment_status", utils.GetLastPaymentStatus(order))
	order.Set("fulfillment_status", utils.GetLastFulfillmentStatus(order))
	return order, nil
}
