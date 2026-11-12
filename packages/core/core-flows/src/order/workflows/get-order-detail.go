package workflows

import (
	"context"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/order/utils"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// GetOrderDetailWorkflowID mirrors getOrderDetailWorkflowId.
const GetOrderDetailWorkflowID = "get-order-detail"

// GetOrderDetailInput mirrors GetOrderDetailWorkflowInput.
type GetOrderDetailInput struct {
	Filters *dal.Record `json:"filters,omitempty"`
	Fields  []string    `json:"fields"`
	OrderID string      `json:"order_id"`
	Version any         `json:"version,omitempty"`
}

// GetOrderDetailWorkflow mirrors getOrderDetailWorkflow: the order (step
// "get-order") with its payment and fulfillment statuses.
var GetOrderDetailWorkflow = composer.CreateWorkflow(GetOrderDetailWorkflowID,
	func(e *composer.Execution, in GetOrderDetailInput) (*dal.Record, error) {
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
		isList := false
		res, err := commonsteps.UseQueryGraphStep.Config("get-order", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
			GraphInput: query.GraphInput{Entity: "order", Fields: fields, Filters: variables},
			Options:    joiner.Options{ThrowIfKeyNotFound: true},
			IsList:     &isList,
		})
		if err != nil {
			return nil, err
		}
		order, _ := res.Data.(*dal.Record)
		if order == nil {
			return nil, nil
		}
		order.Set("payment_status", utils.GetLastPaymentStatus(order))
		order.Set("fulfillment_status", utils.GetLastFulfillmentStatus(order))
		return order, nil
	})

// GetOrderDetail runs getOrderDetailWorkflow (the routes run it like
// upstream: getOrderDetailWorkflow(req.scope).run).
func GetOrderDetail(ctx context.Context, c composer.Container, in GetOrderDetailInput) (*dal.Record, error) {
	return GetOrderDetailWorkflow.Run(ctx, c, in)
}
