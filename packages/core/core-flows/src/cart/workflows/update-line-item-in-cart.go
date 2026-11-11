package workflows

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/steps"
	. "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/utils"
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

var updateItemCartFields = append(append([]string{}, FieldsForPricingContext...), "items.*")

var updateItemVariantFields = Deduplicate(append(append([]string{}, ProductVariantsFields...), "calculated_price.*"), RequiredVariantFieldsForInventoryConfirmation)

// UpdateLineItemInCartInput mirrors UpdateLineItemInCartWorkflowInputDTO.
type UpdateLineItemInCartInput struct {
	CartID         string
	ItemID         string
	Update         *dal.Record
	AdditionalData *dal.Record
}

type lineItemToUpdate struct {
	Item       *dal.Record
	VariantIDs []any
}

var findLineItemToUpdateStep = composer.CreateStep("find-line-item-to-update",
	func(ctx composer.StepExecutionContext, in struct {
		cart   *dal.Record
		itemID string
	}) (composer.StepResponse[lineItemToUpdate], error) {
		for _, i := range Records(Get(in.cart, "items")) {
			if Str(i, "id") == in.itemID {
				ids := []any{}
				if v := Get(i, "variant_id"); Truthy(v) {
					ids = append(ids, v)
				}
				return composer.NewStepResponse(lineItemToUpdate{i, ids}), nil
			}
		}
		return composer.StepResponse[lineItemToUpdate]{}, common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, "Line item with id: "+in.itemID+" was not found")
	}, nil)

var validateVariantPricesStep = composer.CreateStep("validate-variant-prices",
	func(ctx composer.StepExecutionContext, variants []*dal.Record) (composer.StepResponse[any], error) {
		var missing []string
		for _, v := range variants {
			if !IsPresent(Get(Rec(v, "calculated_price"), "calculated_amount")) {
				missing = append(missing, Str(v, "id"))
			}
		}
		if len(missing) > 0 {
			return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Variants with IDs "+joinStrings(missing)+" do not have a price")
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

func joinStrings(l []string) string {
	out := ""
	for i, s := range l {
		if i > 0 {
			out += ", "
		}
		out += s
	}
	return out
}

var prepareLineItemUpdateStep = composer.CreateStep("prepare-line-item-update",
	func(ctx composer.StepExecutionContext, in struct {
		input    UpdateLineItemInCartInput
		variants []*dal.Record
		item     *dal.Record
	}) (composer.StepResponse[*dal.Record], error) {
		var variant *dal.Record
		if len(in.variants) > 0 {
			variant = in.variants[0]
		}
		cp := Rec(variant, "calculated_price")
		data := Copy(in.input.Update)
		if Has(in.input.Update, "unit_price") {
			data.Set("is_custom_price", true)
		} else {
			data.Set("unit_price", Get(in.item, "unit_price"))
			data.Set("is_custom_price", Get(in.item, "is_custom_price"))
		}
		if Truthy(Get(in.item, "is_tax_inclusive")) {
			data.Set("is_tax_inclusive", Get(in.item, "is_tax_inclusive"))
		} else {
			data.Set("is_tax_inclusive", Get(cp, "is_calculated_price_tax_inclusive"))
		}
		if variant != nil && !Truthy(Get(data, "is_custom_price")) {
			data.Set("unit_price", Get(cp, "calculated_amount"))
			if _, ok := in.input.Update.Get("compare_at_unit_price"); !ok {
				isSale := Get(Rec(cp, "calculated_price"), "price_list_type") == "sale"
				if isSale && !totals.Eq(Get(cp, "original_amount"), Get(cp, "calculated_amount")) {
					data.Set("compare_at_unit_price", Get(cp, "original_amount"))
				} else {
					data.Set("compare_at_unit_price", nil)
				}
			}
		}
		if Get(data, "unit_price") == nil {
			return composer.StepResponse[*dal.Record]{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Line item "+Str(in.item, "title")+" has no unit price")
		}
		sel := dal.NewRecord()
		sel.Set("id", in.input.ItemID)
		out := dal.NewRecord()
		out.Set("data", data)
		out.Set("selector", sel)
		return composer.NewStepResponse(out), nil
	}, nil)

type itemsBefore []*dal.Record

var updateLineItemsStepWithSelector = composer.CreateStep("update-line-items-with-selector",
	func(ctx composer.StepExecutionContext, in *dal.Record) (composer.StepResponse[any], error) {
		svc := steps.CartOf(ctx.Container)
		sel, data := Rec(in, "selector"), Rec(in, "data")
		before, err := svc.ListBySuffix(ctx.Context, "LineItems", sel, utilsmodulessdk.FindConfig{Select: append([]string{"id"}, data.Keys()...)})
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		if _, err := commonsteps.UpdateBySelector(ctx.Context, svc, "LineItem", sel, data); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil, any(itemsBefore(before))), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		before, _ := data.(itemsBefore)
		svc := steps.CartOf(ctx.Container)
		for _, i := range before {
			u := dal.NewRecord()
			u.Set("id", Get(i, "id"))
			for _, k := range []string{"quantity", "title", "metadata", "unit_price"} {
				if v, ok := i.Get(k); ok {
					u.Set(k, v)
				}
			}
			if _, err := svc.Internal("LineItem").Update(ctx.Context, []any{u}); err != nil {
				return err
			}
		}
		return nil
	})

// UpdateLineItemInCartWorkflow mirrors updateLineItemInCartWorkflow.
var UpdateLineItemInCartWorkflow = composer.CreateWorkflow("update-line-item-in-cart",
	func(e *composer.Execution, in UpdateLineItemInCartInput) (any, error) {
		if err := acquireLock(e, in.CartID, 2, 10); err != nil {
			return nil, err
		}
		cart, err := fetchCart(e, "get-cart", in.CartID, updateItemCartFields, true)
		if err != nil {
			return nil, err
		}
		found, err := findLineItemToUpdateStep.Run(e, struct {
			cart   *dal.Record
			itemID string
		}{cart, in.ItemID})
		if err != nil {
			return nil, err
		}
		if _, err := steps.ValidateCartStep.Run(e, cart); err != nil {
			return nil, err
		}
		pricing := FilterObjectByKeys(cart, FieldsForPricingContext)
		pricing.Set("quantity", Coalesce(Get(in.Update, "quantity"), Get(found.Item, "quantity")))
		pricing.Set("currency_code", Get(cart, "currency_code"))
		pricing.Set("region_id", Get(cart, "region_id"))
		pricing.Set("region", Get(cart, "region"))
		pricing.Set("customer_id", Get(cart, "customer_id"))
		pricing.Set("customer", Get(cart, "customer"))
		remove := Has(in.Update, "quantity") && totals.Eq(Get(in.Update, "quantity"), 0)
		if remove {
			if _, err := DeleteLineItemsWorkflow.RunAsStep(e, DeleteLineItemsInput{CartID: in.CartID, IDs: []any{in.ItemID}, AdditionalData: in.AdditionalData}); err != nil {
				return nil, err
			}
		}
		var variants []*dal.Record
		if len(found.VariantIDs) > 0 && !remove {
			f := dal.NewRecord()
			f.Set("id", found.VariantIDs)
			qc := dal.NewRecord()
			qc.Set("calculated_price", query.QueryContext(pricing))
			res, err := commonsteps.UseQueryGraphStep.Config("fetch-variants", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{GraphInput: query.GraphInput{Entity: "variants", Fields: updateItemVariantFields, Filters: f, Context: qc}})
			if err != nil {
				return nil, err
			}
			variants = Records(res.Data)
			if _, err := validateVariantPricesStep.Run(e, variants); err != nil {
				return nil, err
			}
		}
		if !remove {
			found.Item.Set("quantity", Get(in.Update, "quantity"))
			if _, err := ConfirmVariantInventoryWorkflow.RunAsStep(e, ConfirmVariantInventoryInput{SalesChannelID: Str(pricing, "sales_channel_id"), Variants: variants, Items: []*dal.Record{found.Item}}); err != nil {
				return nil, err
			}
			update, err := prepareLineItemUpdateStep.Run(e, struct {
				input    UpdateLineItemInCartInput
				variants []*dal.Record
				item     *dal.Record
			}{in, variants, found.Item})
			if err != nil {
				return nil, err
			}
			if _, err := updateLineItemsStepWithSelector.Run(e, update); err != nil {
				return nil, err
			}
			if _, err := RefreshCartItemsWorkflow.RunAsStep(e, RefreshCartItemsInput{CartID: in.CartID, AdditionalData: in.AdditionalData}); err != nil {
				return nil, err
			}
		}
		ev := dal.NewRecord()
		ev.Set("id", in.CartID)
		err = composer.Parallelize(
			func() error { return releaseLock(e, in.CartID) },
			func() error {
				_, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "cart.updated", Data: ev})
				return err
			},
		)
		return nil, err
	})

// DeleteLineItemsInput mirrors DeleteLineItemsWorkflowInput.
type DeleteLineItemsInput struct {
	CartID         string
	IDs            []any
	AdditionalData *dal.Record
}

var deleteLineItemsStep = composer.CreateStep("delete-line-items",
	func(ctx composer.StepExecutionContext, ids []any) (composer.StepResponse[any], error) {
		if _, err := steps.CartOf(ctx.Container).Internal("LineItem").SoftDelete(ctx.Context, ids); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil, any(ids)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		ids, _ := data.([]any)
		if len(ids) == 0 {
			return nil
		}
		_, err := steps.CartOf(ctx.Container).Internal("LineItem").Restore(ctx.Context, ids)
		return err
	})

// DeleteLineItemsWorkflow mirrors deleteLineItemsWorkflow (line-item/workflows).
var DeleteLineItemsWorkflow = composer.CreateWorkflow("delete-line-items",
	func(e *composer.Execution, in DeleteLineItemsInput) (any, error) {
		if err := acquireLock(e, in.CartID, 2, 10); err != nil {
			return nil, err
		}
		if _, err := deleteLineItemsStep.Run(e, in.IDs); err != nil {
			return nil, err
		}
		err := composer.Parallelize(
			func() error {
				_, err := RefreshCartItemsWorkflow.RunAsStep(e, RefreshCartItemsInput{CartID: in.CartID, AdditionalData: in.AdditionalData})
				return err
			},
			func() error {
				ev := dal.NewRecord()
				ev.Set("id", in.CartID)
				_, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "cart.updated", Data: ev})
				return err
			},
		)
		if err != nil {
			return nil, err
		}
		return nil, releaseLock(e, in.CartID)
	})
