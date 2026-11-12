package modulesdiff

import (
	"context"
	"errors"
	"testing"

	ordersteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/order/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// Compensations of steps that end their upstream workflows (so no HTTP
// scenario reaches them): a workflow runs the step and then fails, and the
// data must be back to its state before the step.

type mapContainer map[string]any

func (c mapContainer) Resolve(key string) any { return c[key] }

var errAfterStep = errors.New("failure after the step")

var failStep = composer.CreateStep("test-fail-after-step",
	func(ctx composer.StepExecutionContext, _ any) (composer.StepResponse[any], error) {
		return composer.StepResponse[any]{}, errAfterStep
	}, nil)

type orderServiceForTests interface {
	Internal(className string) *utilsmodulessdk.InternalService
	CreateOrders(ctx context.Context, data []*dal.Record) ([]*dal.Record, error)
	CreateOrderChange(ctx context.Context, data []*dal.Record) ([]*dal.Record, error)
	SetOrderLineItemTaxLines(ctx context.Context, orderID string, taxLines []*dal.Record) ([]*dal.Record, error)
}

func rec(kv ...any) *dal.Record {
	r := dal.NewRecord()
	for i := 0; i+1 < len(kv); i += 2 {
		r.Set(kv[i].(string), kv[i+1])
	}
	return r
}

func field(r *dal.Record, k string) any {
	v, _ := r.Get(k)
	return v
}

func TestStepCompensations(t *testing.T) {
	pool := setupDatabase(t, "medusa_diff_go_compensations")
	ctx := context.Background()
	inst, err := moduleServiceInstance(pool, "order")
	if err != nil {
		t.Fatal(err)
	}
	svc := inst.(orderServiceForTests)
	c := mapContainer{utilsmodulessdk.ORDER: inst}

	orders, err := svc.CreateOrders(ctx, []*dal.Record{rec(
		"currency_code", "eur", "email", "buyer@example.test",
		"items", []any{rec("title", "Shirt", "quantity", 1, "unit_price", 10), rec("title", "Pants", "quantity", 1, "unit_price", 20)},
	)})
	if err != nil {
		t.Fatal(err)
	}
	order := orders[0]
	orderID := field(order, "id").(string)
	items, err := svc.Internal("OrderLineItem").List(ctx, nil, utilsmodulessdk.FindConfig{})
	if err != nil || len(items) != 2 {
		t.Fatalf("items = %d, %v", len(items), err)
	}
	item1, item2 := field(items[0], "id"), field(items[1], "id")
	if _, err := svc.SetOrderLineItemTaxLines(ctx, orderID, []*dal.Record{
		rec("item_id", item1, "code", "OLD1", "rate", 10, "description", "old 1"),
		rec("item_id", item2, "code", "OLD2", "rate", 10, "description", "old 2"),
	}); err != nil {
		t.Fatal(err)
	}
	codes := func() map[any][]any {
		lines, err := svc.Internal("OrderLineItemTaxLine").List(ctx, nil, utilsmodulessdk.FindConfig{})
		if err != nil {
			t.Fatal(err)
		}
		out := map[any][]any{}
		for _, l := range lines {
			out[field(l, "item_id")] = append(out[field(l, "item_id")], field(l, "code"))
		}
		return out
	}

	t.Run("set-order-tax-lines-for-items", func(t *testing.T) {
		for _, full := range []bool{true, false} {
			wf := composer.CreateWorkflow("test-revert-order-tax-lines", func(e *composer.Execution, in ordersteps.SetOrderTaxLinesInput) (any, error) {
				if _, err := ordersteps.SetOrderTaxLinesForItemsStep.Run(e, in); err != nil {
					return nil, err
				}
				return failStep.Run(e, nil)
			})
			orderWithItems := rec("id", orderID, "items", []any{rec("id", item1), rec("id", item2)})
			_, err := wf.Run(ctx, c, ordersteps.SetOrderTaxLinesInput{
				Order:             orderWithItems,
				ItemTaxLines:      []*dal.Record{rec("line_item_id", item1, "code", "NEW", "rate", 20, "name", "new")},
				IsFullReplacement: full,
			})
			if !errors.Is(err, errAfterStep) {
				t.Fatalf("full=%v: err = %v", full, err)
			}
			got := codes()
			if len(got[item1]) != 1 || got[item1][0] != "OLD1" || len(got[item2]) != 1 || got[item2][0] != "OLD2" {
				t.Fatalf("full=%v: tax lines after the compensation = %v", full, got)
			}
		}
	})

	t.Run("decline-order-change", func(t *testing.T) {
		changes, err := svc.CreateOrderChange(ctx, []*dal.Record{rec("order_id", orderID, "description", "edit")})
		if err != nil {
			t.Fatal(err)
		}
		changeID := field(changes[0], "id")
		wf := composer.CreateWorkflow("test-revert-decline-order-change", func(e *composer.Execution, in *dal.Record) (any, error) {
			if _, err := ordersteps.DeclineOrderChangeStep.Run(e, in); err != nil {
				return nil, err
			}
			return failStep.Run(e, nil)
		})
		_, err = wf.Run(ctx, c, rec("id", changeID, "declined_by", "user_1", "declined_reason", "no"))
		if !errors.Is(err, errAfterStep) {
			t.Fatalf("err = %v", err)
		}
		after, err := svc.Internal("OrderChange").Retrieve(ctx, changeID, utilsmodulessdk.FindConfig{})
		if err != nil {
			t.Fatal(err)
		}
		// Upstream restores the fields it read before the decline (the input
		// fields, order_id, return/claim/exchange ids, version and the
		// declined_* fields); status is not among them and stays "declined".
		if field(after, "status") != "declined" || field(after, "declined_at") != nil || field(after, "declined_by") != nil || field(after, "declined_reason") != nil {
			t.Fatalf("order change after the compensation: status=%v declined_at=%v declined_by=%v declined_reason=%v",
				field(after, "status"), field(after, "declined_at"), field(after, "declined_by"), field(after, "declined_reason"))
		}
	})
}
