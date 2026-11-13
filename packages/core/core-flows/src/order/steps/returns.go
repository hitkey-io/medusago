package steps

import (
	"context"
	"fmt"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

type returnOps interface {
	Internal(className string) *utilsmodulessdk.InternalService
	CreateReturns(ctx context.Context, data []*dal.Record) ([]*dal.Record, error)
	CreateReturnItems(ctx context.Context, data []*dal.Record) ([]*dal.Record, error)
	DeleteReturnItems(ctx context.Context, ids []any) error
	CreateOrderShippingMethods(ctx context.Context, data []*dal.Record) ([]*dal.Record, error)
	SoftDeleteOrderShippingMethods(ctx context.Context, ids []any) (any, error)
	RestoreOrderShippingMethods(ctx context.Context, ids []any) error
	PreviewOrderChange(ctx context.Context, orderID string) (*dal.Record, error)
	UpsertOrderShippingMethodTaxLines(ctx context.Context, taxLines []*dal.Record) ([]*dal.Record, error)
	UpsertOrderLineItemTaxLines(ctx context.Context, taxLines []*dal.Record) ([]*dal.Record, error)
	SetOrderShippingMethodTaxLines(ctx context.Context, orderID string, taxLines []*dal.Record) ([]*dal.Record, error)
	SetOrderLineItemTaxLines(ctx context.Context, orderID string, taxLines []*dal.Record) ([]*dal.Record, error)
}

func returnsOf(c composer.Container) returnOps { return c.Resolve(utilsmodulessdk.ORDER).(returnOps) }

// CreateReturnsStep mirrors createReturnsStep.
var CreateReturnsStep = composer.CreateStep("create-returns",
	func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		created, err := returnsOf(ctx.Container).CreateReturns(ctx.Context, data)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(created, any(ids(created))), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		list, _ := data.([]any)
		if len(list) == 0 {
			return nil
		}
		_, err := returnsOf(ctx.Container).Internal("Return").Delete(ctx.Context, list)
		return err
	})

// CreateReturnItemsFromActionsInput mirrors CreateReturnItemsFromActionsInput.
type CreateReturnItemsFromActionsInput struct {
	Changes  []*dal.Record
	ReturnID any
}

// CreateReturnItemsFromActionsStep mirrors createReturnItemsFromActionsStep.
var CreateReturnItemsFromActionsStep = composer.CreateStep("create-return-items-from-change-actions",
	func(ctx composer.StepExecutionContext, in CreateReturnItemsFromActionsInput) (composer.StepResponse[[]*dal.Record], error) {
		items := make([]*dal.Record, 0, len(in.Changes))
		for _, c := range in.Changes {
			details, _ := get(c, "details").(*dal.Record)
			r := dal.NewRecord()
			r.Set("return_id", in.ReturnID)
			r.Set("item_id", get(details, "reference_id"))
			if details != nil {
				if v, ok := details.Get("reason_id"); ok {
					r.Set("reason_id", v)
				}
			}
			r.Set("quantity", get(details, "quantity"))
			if v, ok := c.Get("internal_note"); ok {
				r.Set("note", v)
			}
			if m := get(details, "metadata"); m != nil {
				r.Set("metadata", m)
			} else {
				r.Set("metadata", dal.NewRecord())
			}
			items = append(items, r)
		}
		created, err := returnsOf(ctx.Container).CreateReturnItems(ctx.Context, items)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(created, any(ids(created))), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		list, _ := data.([]any)
		if list == nil {
			return nil
		}
		return returnsOf(ctx.Container).DeleteReturnItems(ctx.Context, list)
	})

func updateEntitiesStep(name, className, suffix string) *composer.Step[[]*dal.Record, []*dal.Record] {
	return composer.CreateStep(name,
		func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
			svc := returnsOf(ctx.Container)
			f := dal.NewRecord()
			f.Set("id", ids(data))
			sel := []string{"id"}
			seen := map[string]bool{"id": true}
			for _, d := range data {
				for _, k := range d.Keys() {
					if !seen[k] {
						seen[k] = true
						sel = append(sel, k)
					}
				}
			}
			prev, err := svc.Internal(className).List(ctx.Context, f, utilsmodulessdk.FindConfig{Select: sel})
			if err != nil {
				return composer.StepResponse[[]*dal.Record]{}, err
			}
			in := make([]any, len(data))
			for i, d := range data {
				in[i] = d
			}
			res, err := svc.Internal(className).Update(ctx.Context, in)
			if err != nil {
				return composer.StepResponse[[]*dal.Record]{}, err
			}
			out, _ := res.([]*dal.Record)
			return composer.NewStepResponse(out, any(prev)), nil
		},
		func(ctx composer.StepExecutionContext, data any) error {
			prev, _ := data.([]*dal.Record)
			if len(prev) == 0 {
				return nil
			}
			in := make([]any, len(prev))
			for i, d := range prev {
				in[i] = d
			}
			_, err := returnsOf(ctx.Container).Internal(className).Update(ctx.Context, in)
			return err
		})
}

// UpdateReturnsStep mirrors updateReturnsStep.
var UpdateReturnsStep = updateEntitiesStep("update-returns", "Return", "Returns")

// UpdateReturnItemsStep mirrors updateReturnItemsStep.
var UpdateReturnItemsStep = updateEntitiesStep("update-return-items", "ReturnItem", "ReturnItems")

func softDeleteStep(name, className string) *composer.Step[[]any, any] {
	return composer.CreateStep(name,
		func(ctx composer.StepExecutionContext, in []any) (composer.StepResponse[any], error) {
			var list []any
			for _, id := range in {
				if id != nil && id != "" {
					list = append(list, id)
				}
			}
			// ids.length ? service.softDelete<X>(ids) : []
			var deleted any = []any{}
			if len(list) > 0 {
				var err error
				if deleted, err = commonsteps.SoftDeleteVia(ctx.Context, returnsOf(ctx.Container), "softDelete"+utilsmodulessdk.Pluralize(className), className, list); err != nil {
					return composer.StepResponse[any]{}, err
				}
			}
			return composer.NewStepResponse[any](deleted, any(in)), nil
		},
		func(ctx composer.StepExecutionContext, data any) error {
			list, _ := data.([]any)
			if len(list) == 0 {
				return nil
			}
			_, err := returnsOf(ctx.Container).Internal(className).Restore(ctx.Context, list)
			return err
		})
}

// DeleteReturnsStep mirrors deleteReturnsStep.
var DeleteReturnsStep = softDeleteStep("delete-return", "Return")

// DeleteOrderChangesStep mirrors deleteOrderChangesStep.
var DeleteOrderChangesStep = softDeleteStep("delete-order-change", "OrderChange")

// DeleteOrderShippingMethodsStep mirrors deleteOrderShippingMethods.
var DeleteOrderShippingMethodsStep = composer.CreateStep("delete-order-shipping-methods",
	func(ctx composer.StepExecutionContext, in []any) (composer.StepResponse[any], error) {
		deleted, err := returnsOf(ctx.Container).SoftDeleteOrderShippingMethods(ctx.Context, in)
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](deleted, any(in)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		list, _ := data.([]any)
		if list == nil {
			return nil
		}
		return returnsOf(ctx.Container).RestoreOrderShippingMethods(ctx.Context, list)
	})

// PreviewOrderChangeStep mirrors previewOrderChangeStep.
var PreviewOrderChangeStep = composer.CreateStep("preview-order-change",
	func(ctx composer.StepExecutionContext, orderID string) (composer.StepResponse[*dal.Record], error) {
		preview, err := returnsOf(ctx.Container).PreviewOrderChange(ctx.Context, orderID)
		if err != nil {
			return composer.StepResponse[*dal.Record]{}, err
		}
		return composer.NewStepResponse(preview), nil
	}, nil)

// CreateOrderShippingMethodsStep mirrors createOrderShippingMethods.
var CreateOrderShippingMethodsStep = composer.CreateStep("create-order-shipping-methods",
	func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		created, err := returnsOf(ctx.Container).CreateOrderShippingMethods(ctx.Context, data)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(created, any(ids(created))), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		list, _ := data.([]any)
		if list == nil {
			return nil
		}
		_, err := returnsOf(ctx.Container).Internal("OrderShippingMethod").Delete(ctx.Context, list)
		return err
	})

// SetOrderTaxLinesInput mirrors SetOrderTaxLinesForItemsStepInput.
type SetOrderTaxLinesInput struct {
	Order             *dal.Record
	ItemTaxLines      []*dal.Record
	ShippingTaxLines  []*dal.Record
	IsFullReplacement bool
}

func normalizeTaxLines(lines []*dal.Record, refKey, targetKey string) []*dal.Record {
	out := make([]*dal.Record, 0, len(lines))
	for _, l := range lines {
		r := dal.NewRecord()
		r.Set("description", get(l, "name"))
		r.Set("tax_rate_id", get(l, "rate_id"))
		r.Set("code", get(l, "code"))
		r.Set("rate", get(l, "rate"))
		r.Set("provider_id", get(l, "provider_id"))
		r.Set(targetKey, get(l, refKey))
		if v, ok := l.Get("data"); ok {
			r.Set("data", v)
		}
		out = append(out, r)
	}
	return out
}

// SetOrderTaxLinesForItemsStep mirrors setOrderTaxLinesForItemsStep: a full
// replacement with computed tax lines replaces the order's tax lines, other
// computed tax lines are upserted (inserted: they carry no id), no computed
// tax lines change nothing. The tax lines of the affected items and shipping
// methods are captured first; the compensation restores them.
var SetOrderTaxLinesForItemsStep = composer.CreateStep("set-order-tax-lines-for-items",
	func(ctx composer.StepExecutionContext, in SetOrderTaxLinesInput) (composer.StepResponse[any], error) {
		svc := returnsOf(ctx.Container)
		items := normalizeTaxLines(in.ItemTaxLines, "line_item_id", "item_id")
		shipping := normalizeTaxLines(in.ShippingTaxLines, "shipping_line_id", "shipping_method_id")
		orderID, _ := get(in.Order, "id").(string)

		taxLinesOp := func(lines []*dal.Record) string {
			switch {
			case in.IsFullReplacement && len(lines) > 0:
				return "replace"
			case len(lines) > 0:
				return "upsert"
			}
			return "none"
		}
		itemsOp, shippingOp := taxLinesOp(items), taxLinesOp(shipping)

		affected := func(op, relation string, lines []*dal.Record, refKey string) []any {
			out := []any{}
			if rel := records(get(in.Order, relation)); op == "replace" && len(rel) > 0 {
				for _, r := range rel {
					out = append(out, get(r, "id"))
				}
				return out
			}
			for _, l := range lines {
				out = append(out, get(l, refKey))
			}
			return out
		}
		affectedItemIDs := affected(itemsOp, "items", in.ItemTaxLines, "line_item_id")
		affectedShippingIDs := affected(shippingOp, "shipping_methods", in.ShippingTaxLines, "shipping_line_id")

		existing := func(className, key string, ids []any) ([]*dal.Record, error) {
			if len(ids) == 0 {
				return []*dal.Record{}, nil
			}
			f := dal.NewRecord()
			f.Set(key, ids)
			return svc.Internal(className).List(ctx.Context, f, utilsmodulessdk.FindConfig{})
		}
		existingShipping, err := existing("OrderShippingMethodTaxLine", "shipping_method_id", affectedShippingIDs)
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		existingItems, err := existing("OrderLineItemTaxLine", "item_id", affectedItemIDs)
		if err != nil {
			return composer.StepResponse[any]{}, err
		}

		switch itemsOp {
		case "replace":
			_, err = svc.SetOrderLineItemTaxLines(ctx.Context, orderID, items)
		case "upsert":
			_, err = svc.UpsertOrderLineItemTaxLines(ctx.Context, items)
		}
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		switch shippingOp {
		case "replace":
			_, err = svc.SetOrderShippingMethodTaxLines(ctx.Context, orderID, shipping)
		case "upsert":
			_, err = svc.UpsertOrderShippingMethodTaxLines(ctx.Context, shipping)
		}
		if err != nil {
			return composer.StepResponse[any]{}, err
		}

		revert := dal.NewRecord()
		revert.Set("order", in.Order)
		revert.Set("existingLineItemTaxLines", existingItems)
		revert.Set("existingShippingMethodTaxLines", existingShipping)
		revert.Set("affectedItemIds", affectedItemIDs)
		revert.Set("affectedShippingIds", affectedShippingIDs)
		revert.Set("itemsOp", itemsOp)
		revert.Set("shippingOp", shippingOp)
		return composer.NewStepResponse[any](nil, any(revert)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		revert, _ := data.(*dal.Record)
		if revert == nil {
			return nil
		}
		svc := returnsOf(ctx.Container)
		orderID, _ := get(asRec(get(revert, "order")), "id").(string)

		snapshot := func(lines any, refKey string) []*dal.Record {
			out := []*dal.Record{}
			for _, l := range records(lines) {
				r := dal.NewRecord()
				for _, k := range []string{"description", "tax_rate_id", "code", "rate", "provider_id", refKey, "metadata", "data"} {
					r.Set(k, get(l, k))
				}
				out = append(out, r)
			}
			return out
		}
		itemTaxLines := snapshot(get(revert, "existingLineItemTaxLines"), "item_id")
		shippingTaxLines := snapshot(get(revert, "existingShippingMethodTaxLines"), "shipping_method_id")

		restore := func(op, className, key string, affectedIDs any, lines []*dal.Record, set func() error) error {
			switch op {
			case "replace":
				// whole order restore: also deletes the lines the step set
				return set()
			case "upsert":
				// the upsert only inserted lines: the affected entities' lines
				// are deleted and the captured ones recreated
				f := dal.NewRecord()
				f.Set(key, affectedIDs)
				if _, err := svc.Internal(className).Delete(ctx.Context, f); err != nil {
					return err
				}
				if len(lines) > 0 {
					_, err := svc.Internal(className).Create(ctx.Context, lines)
					return err
				}
			}
			return nil
		}
		if err := restore(fmt.Sprint(get(revert, "itemsOp")), "OrderLineItemTaxLine", "item_id", get(revert, "affectedItemIds"), itemTaxLines, func() error {
			_, err := svc.SetOrderLineItemTaxLines(ctx.Context, orderID, itemTaxLines)
			return err
		}); err != nil {
			return err
		}
		return restore(fmt.Sprint(get(revert, "shippingOp")), "OrderShippingMethodTaxLine", "shipping_method_id", get(revert, "affectedShippingIds"), shippingTaxLines, func() error {
			_, err := svc.SetOrderShippingMethodTaxLines(ctx.Context, orderID, shippingTaxLines)
			return err
		})
	})

// CancelOrderReturnStep mirrors cancelOrderReturnStep.
var CancelOrderReturnStep = composer.CreateStep("cancel-order-return",
	func(ctx composer.StepExecutionContext, data *dal.Record) (composer.StepResponse[any], error) {
		svc, _ := ctx.Container.Resolve(utilsmodulessdk.ORDER).(interface {
			CancelReturn(ctx context.Context, data *dal.Record) (*dal.Record, error)
		})
		if _, err := svc.CancelReturn(ctx.Context, data); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil, get(data, "order_id")), nil
	}, RevertLastVersion)

func asRec(v any) *dal.Record {
	r, _ := v.(*dal.Record)
	return r
}
