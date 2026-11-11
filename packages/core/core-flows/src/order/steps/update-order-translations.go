package steps

import (
	"context"

	cartutils "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/utils"
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// UpdateOrderItemsTranslationsStepInput mirrors
// UpdateOrderItemsTranslationsStepInput.
type UpdateOrderItemsTranslationsStepInput struct {
	OrderID string
	Locale  string
	// Items are pre-loaded items (nil: the order items are read).
	Items []*dal.Record
}

const orderItemsTranslationsBatchSize = 100

var orderItemsTranslationsLineItemFields = []string{"id", "variant_id", "product_id", "title", "subtitle", "product_title", "product_description", "product_subtitle", "product_type", "product_collection", "product_handle", "variant_title"}

var orderItemTranslationKeys = []string{"title", "subtitle", "product_title", "product_description", "product_subtitle", "product_type", "product_collection", "product_handle", "variant_title"}

// orderItemTranslationSnapshot is { id, title, ..., variant_title } (an
// undefined value stays undefined).
func orderItemTranslationSnapshot(item *dal.Record) *dal.Record {
	r := dal.NewRecord()
	r.Set("id", get(item, "id"))
	for _, k := range orderItemTranslationKeys {
		if v, ok := item.Get(k); ok {
			r.Set(k, v)
		}
	}
	return r
}

type orderInternal interface {
	Internal(className string) *utilsmodulessdk.InternalService
}

func orderInternalOf(c composer.Container) orderInternal {
	return c.Resolve(utilsmodulessdk.ORDER).(orderInternal)
}

// updateOrderLineItemsByID mirrors updateOrderLineItems([{ selector: { id },
// data }]): the matching items are updated (no quantity: the item details
// are left as they are).
func updateOrderLineItemsByID(ctx context.Context, c composer.Container, updates []*dal.Record) error {
	svc := orderInternalOf(c)
	for i := 0; i < len(updates); i += orderItemsTranslationsBatchSize {
		end := min(i+orderItemsTranslationsBatchSize, len(updates))
		var batch []any
		for _, u := range updates[i:end] {
			f := dal.NewRecord()
			f.Set("id", get(u, "id"))
			rows, err := svc.Internal("OrderLineItem").List(ctx, f, utilsmodulessdk.FindConfig{})
			if err != nil {
				return err
			}
			for _, row := range rows {
				r := dal.NewRecord()
				for _, k := range u.Keys() {
					if k == "id" {
						continue
					}
					v, _ := u.Get(k)
					r.Set(k, v)
				}
				r.Set("id", get(row, "id"))
				batch = append(batch, r)
			}
		}
		if len(batch) > 0 {
			if _, err := svc.Internal("OrderLineItem").Update(ctx, batch); err != nil {
				return err
			}
		}
	}
	return nil
}

// UpdateOrderItemsTranslationsStep mirrors updateOrderItemsTranslationsStep:
// the line items of the order are translated in the new locale (the
// translated variant data replaces the product snapshot fields).
var UpdateOrderItemsTranslationsStep = composer.CreateStep("update-order-items-translations",
	func(ctx composer.StepExecutionContext, in UpdateOrderItemsTranslationsStepInput) (composer.StepResponse[any], error) {
		var original []*dal.Record
		q, _ := ctx.Container.Resolve(common.ContainerRegistrationKeys.QUERY).(*query.Query)
		if !q.TranslationsEnabled() || in.Locale == "" {
			return composer.NewStepResponse[any](nil, any([]*dal.Record{})), nil
		}
		processBatch := func(items []*dal.Record) error {
			var variantIDs []any
			seen := map[any]bool{}
			for _, item := range items {
				id := get(item, "variant_id")
				if s, ok := id.(string); ok && s != "" && !seen[id] {
					seen[id] = true
					variantIDs = append(variantIDs, id)
				}
			}
			if len(variantIDs) == 0 {
				return nil
			}
			for _, item := range items {
				original = append(original, orderItemTranslationSnapshot(item))
			}
			f := dal.NewRecord()
			f.Set("id", variantIDs)
			res, err := q.Graph(ctx.Context, query.GraphInput{Entity: "variants", Filters: f, Fields: cartutils.ProductVariantsFields}, joiner.Options{Locale: in.Locale})
			if err != nil {
				return err
			}
			variants := make([]*dal.Record, 0, len(res.Data))
			for _, v := range res.Data {
				if r, ok := v.(*dal.Record); ok {
					variants = append(variants, r)
				}
			}
			copies := make([]*dal.Record, len(items))
			for i, item := range items {
				copies[i] = cartutils.Copy(item)
			}
			var updates []*dal.Record
			for _, item := range commonsteps.ApplyTranslationsToItems(copies, variants) {
				if id := get(item, "id"); id == nil || id == "" {
					continue
				}
				updates = append(updates, orderItemTranslationSnapshot(item))
			}
			if len(updates) > 0 {
				return updateOrderLineItemsByID(ctx.Context, ctx.Container, updates)
			}
			return nil
		}
		fail := func(err error) (composer.StepResponse[any], error) {
			_ = updateOrderLineItemsByID(ctx.Context, ctx.Container, original)
			return composer.StepResponse[any]{}, err
		}
		if len(in.Items) > 0 {
			if err := processBatch(in.Items); err != nil {
				return fail(err)
			}
			return composer.NewStepResponse[any](nil, any(original)), nil
		}
		f := dal.NewRecord()
		f.Set("id", in.OrderID)
		fields := make([]string, len(orderItemsTranslationsLineItemFields))
		for i, k := range orderItemsTranslationsLineItemFields {
			fields[i] = "items." + k
		}
		res, err := q.Graph(ctx.Context, query.GraphInput{Entity: "orders", Filters: f, Fields: fields}, joiner.Options{})
		if err != nil {
			return fail(err)
		}
		var items []*dal.Record
		if len(res.Data) > 0 {
			if order, ok := res.Data[0].(*dal.Record); ok {
				items = records(get(order, "items"))
			}
		}
		for i := 0; i < len(items); i += orderItemsTranslationsBatchSize {
			end := min(i+orderItemsTranslationsBatchSize, len(items))
			if err := processBatch(items[i:end]); err != nil {
				return fail(err)
			}
		}
		return composer.NewStepResponse[any](nil, any(original)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		original, _ := data.([]*dal.Record)
		if len(original) == 0 {
			return nil
		}
		return updateOrderLineItemsByID(ctx.Context, ctx.Container, original)
	})

// UpdateOrderShippingMethodsTranslationsStepInput mirrors
// UpdateOrderShippingMethodsTranslationsStepInput.
type UpdateOrderShippingMethodsTranslationsStepInput struct {
	ShippingMethods []*dal.Record
	Locale          string
}

// UpdateOrderShippingMethodsTranslationsStep mirrors
// updateOrderShippingMethodsTranslationsStep: the name of the shipping
// methods with a shipping option becomes the translated option name.
var UpdateOrderShippingMethodsTranslationsStep = composer.CreateStep("update-order-shipping-methods-translations",
	func(ctx composer.StepExecutionContext, in UpdateOrderShippingMethodsTranslationsStepInput) (composer.StepResponse[[]*dal.Record], error) {
		q, _ := ctx.Container.Resolve(common.ContainerRegistrationKeys.QUERY).(*query.Query)
		if !q.TranslationsEnabled() || in.Locale == "" || len(in.ShippingMethods) == 0 {
			return composer.NewStepResponse(in.ShippingMethods), nil
		}
		ids := make([]any, len(in.ShippingMethods))
		for i, sm := range in.ShippingMethods {
			ids[i] = get(sm, "shipping_option_id")
		}
		f := dal.NewRecord()
		f.Set("id", ids)
		res, err := q.Graph(ctx.Context, query.GraphInput{Entity: "shipping_option", Fields: []string{"id", "name"}, Filters: f}, joiner.Options{Locale: in.Locale})
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		names := map[any]any{}
		for _, x := range res.Data {
			if r, ok := x.(*dal.Record); ok {
				names[get(r, "id")] = get(r, "name")
			}
		}
		updates := make([]any, len(in.ShippingMethods))
		for i, sm := range in.ShippingMethods {
			u := cartutils.Copy(sm)
			if so := get(sm, "shipping_option_id"); so != nil && so != "" {
				// Map.get of a missing option is undefined (not written)
				if name, ok := names[so]; ok {
					u.Set("name", name)
				} else {
					u.Delete("name")
				}
			}
			updates[i] = u
		}
		updated, err := orderInternalOf(ctx.Container).Internal("OrderShippingMethod").Update(ctx.Context, updates)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(records(updated), any(in.ShippingMethods)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		before, _ := data.([]*dal.Record)
		if len(before) == 0 {
			return nil
		}
		list := make([]any, len(before))
		for i, r := range before {
			list[i] = r
		}
		_, err := orderInternalOf(ctx.Container).Internal("OrderShippingMethod").Update(ctx.Context, list)
		return err
	})

// UpdateOrderTaxLinesTranslationsStepInput mirrors
// UpdateOrderTaxLinesTranslationsStepInput.
type UpdateOrderTaxLinesTranslationsStepInput struct {
	OrderID string
	Locale  string
}

// UpdateOrderTaxLinesTranslationsStep mirrors
// updateOrderTaxLinesTranslationsStep: the description of the tax lines is
// the translated name of their tax rate.
var UpdateOrderTaxLinesTranslationsStep = composer.CreateStep("update-order-tax-lines-translations",
	func(ctx composer.StepExecutionContext, in UpdateOrderTaxLinesTranslationsStepInput) (composer.StepResponse[any], error) {
		q, _ := ctx.Container.Resolve(common.ContainerRegistrationKeys.QUERY).(*query.Query)
		if !q.TranslationsEnabled() || in.Locale == "" {
			return composer.NewStepResponse[any](nil, any([]any{})), nil
		}
		f := dal.NewRecord()
		f.Set("id", in.OrderID)
		res, err := q.Graph(ctx.Context, query.GraphInput{Entity: "order", Filters: f, Fields: []string{
			"items.tax_lines.id", "items.tax_lines.tax_rate_id", "items.tax_lines.description",
			"shipping_methods.tax_lines.id", "shipping_methods.tax_lines.tax_rate_id", "shipping_methods.tax_lines.description",
		}}, joiner.Options{})
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		var order *dal.Record
		if len(res.Data) > 0 {
			order, _ = res.Data[0].(*dal.Record)
		}
		var itemTaxLines, shippingTaxLines []*dal.Record
		for _, item := range records(get(order, "items")) {
			itemTaxLines = append(itemTaxLines, records(get(item, "tax_lines"))...)
		}
		for _, sm := range records(get(order, "shipping_methods")) {
			shippingTaxLines = append(shippingTaxLines, records(get(sm, "tax_lines"))...)
		}
		translate := func(lines []*dal.Record) ([]*dal.Record, error) {
			rates := make([]any, len(lines))
			for i, l := range lines {
				r := dal.NewRecord()
				r.Set("id", get(l, "tax_rate_id"))
				r.Set("name", get(l, "description"))
				r.Set("tax_line_id", get(l, "id"))
				rates[i] = r
			}
			if err := q.ApplyTranslations(ctx.Context, in.Locale, rates); err != nil {
				return nil, err
			}
			out := make([]*dal.Record, len(rates))
			for i, x := range rates {
				r := x.(*dal.Record)
				u := dal.NewRecord()
				u.Set("id", get(r, "tax_line_id"))
				u.Set("description", get(r, "name"))
				out[i] = u
			}
			return out, nil
		}
		items, err := translate(itemTaxLines)
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		shipping, err := translate(shippingTaxLines)
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		svc := returnsOf(ctx.Container)
		if _, err := svc.UpsertOrderLineItemTaxLines(ctx.Context, items); err != nil {
			return composer.StepResponse[any]{}, err
		}
		if _, err := svc.UpsertOrderShippingMethodTaxLines(ctx.Context, shipping); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil, any([][]*dal.Record{itemTaxLines, shippingTaxLines})), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		pair, _ := data.([][]*dal.Record)
		if len(pair) != 2 {
			return nil
		}
		restore := func(lines []*dal.Record) []*dal.Record {
			out := make([]*dal.Record, len(lines))
			for i, l := range lines {
				u := dal.NewRecord()
				u.Set("id", get(l, "id"))
				u.Set("description", get(l, "description"))
				out[i] = u
			}
			return out
		}
		svc := returnsOf(ctx.Container)
		if _, err := svc.UpsertOrderLineItemTaxLines(ctx.Context, restore(pair[0])); err != nil {
			return err
		}
		_, err := svc.UpsertOrderShippingMethodTaxLines(ctx.Context, restore(pair[1]))
		return err
	})

func records(v any) []*dal.Record {
	switch t := v.(type) {
	case []*dal.Record:
		return t
	case []any:
		out := make([]*dal.Record, 0, len(t))
		for _, x := range t {
			if r, ok := x.(*dal.Record); ok {
				out = append(out, r)
			}
		}
		return out
	}
	return nil
}
