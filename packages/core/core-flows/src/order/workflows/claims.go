package workflows

import (
	"fmt"
	"strings"
	"time"

	cartsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/steps"
	cartutils "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/utils"
	cartworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/workflows"
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	fulfillmentworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/fulfillment/workflows"
	ordersteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/order/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// FieldsToComputeAdjustmentsForPreview mirrors fieldsToComputeAdjustmentsForPreview.
var FieldsToComputeAdjustmentsForPreview = []string{"id", "currency_code", "email", "sales_channel_id", "region_id", "customer_id", "customer.id",
	"customer.groups.id", "shipping_address.country_code", "items.id", "items.product_id", "items.quantity", "items.subtotal", "items.original_total",
	"items.is_discountable", "items.adjustments.id", "items.adjustments.code", "shipping_methods.id", "shipping_methods.subtotal",
	"shipping_methods.original_total", "shipping_methods.adjustments.id", "shipping_methods.adjustments.code", "shipping_methods.shipping_option_id",
	"promotions.id", "promotions.code"}

// ComputeAdjustmentsForPreviewInput mirrors ComputeAdjustmentsForPreviewWorkflowInput.
type ComputeAdjustmentsForPreviewInput struct {
	Order       *dal.Record
	OrderChange *dal.Record
}

// ComputeAdjustmentsForPreviewWorkflow mirrors computeAdjustmentsForPreviewWorkflow
// (carried over promotions are not supported yet).
var ComputeAdjustmentsForPreviewWorkflow = composer.CreateWorkflow("compute-adjustments-for-preview",
	func(e *composer.Execution, in ComputeAdjustmentsForPreviewInput) (any, error) {
		previewed, err := ordersteps.PreviewOrderChangeStep.Run(e, fmt.Sprint(g(in.Order, "id")))
		if err != nil {
			return nil, err
		}
		hookIn := dal.NewRecord()
		hookIn.Set("order", in.Order)
		hookIn.Set("orderChange", in.OrderChange)
		hookIn.Set("previewedOrder", previewed)
		if err := composer.CreateHook(e, "setPromotionContext", hookIn); err != nil {
			return nil, err
		}
		if len(recs(g(in.Order, "promotions"))) > 0 && truthy(g(in.OrderChange, "carry_over_promotions")) {
			return nil, common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, "Carrying over promotions to order changes is not supported yet")
		}
		if !truthy(g(rec(g(previewed, "order_change")), "carry_over_promotions")) {
			itemIDs, err := ordersteps.ListOrderChangeActionsByTypeStep.Config("list-item-adjustment-actions-for-deletion", composer.StepOptions{}).Run(e,
				ordersteps.ListOrderChangeActionsByTypeInput{OrderChangeID: g(in.OrderChange, "id"), ActionType: "ITEM_ADJUSTMENTS_REPLACE"})
			if err != nil {
				return nil, err
			}
			shippingIDs, err := ordersteps.ListOrderChangeActionsByTypeStep.Config("list-shipping-adjustment-actions-for-deletion", composer.StepOptions{}).Run(e,
				ordersteps.ListOrderChangeActionsByTypeInput{OrderChangeID: g(in.OrderChange, "id"), ActionType: "SHIPPING_ADJUSTMENTS_REPLACE"})
			if err != nil {
				return nil, err
			}
			all := append(append([]any{}, itemIDs...), shippingIDs...)
			if _, err := ordersteps.DeleteOrderChangeActionsStep.Run(e, all); err != nil {
				return nil, err
			}
		}
		return nil, nil
	})

func withPromotions(order *dal.Record) *dal.Record {
	o := dal.NewRecord()
	for _, k := range order.Keys() {
		v, _ := order.Get(k)
		o.Set(k, v)
	}
	if g(order, "promotions") == nil {
		o.Set("promotions", []any{})
	}
	return o
}

// AddOrderLineItemsInput mirrors OrderAddLineItemWorkflowInput.
type AddOrderLineItemsInput struct {
	OrderID any
	Items   []*dal.Record
}

// AddOrderLineItemsWorkflow mirrors addOrderLineItemsWorkflow.
var AddOrderLineItemsWorkflow = composer.CreateWorkflow("order-add-line-items",
	func(e *composer.Execution, in AddOrderLineItemsInput) ([]*dal.Record, error) {
		f := dal.NewRecord()
		f.Set("id", in.OrderID)
		isList := false
		res, err := commonsteps.UseQueryGraphStep.Config("order-query", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
			GraphInput: query.GraphInput{Entity: "order", Fields: cartutils.FieldsForPricingContext, Filters: f},
			Options:    joiner.Options{ThrowIfKeyNotFound: true},
			IsList:     &isList,
		})
		if err != nil {
			return nil, err
		}
		order := rec(res.Data)
		var variantIDs []any
		for _, item := range in.Items {
			if v := g(item, "variant_id"); truthy(v) {
				variantIDs = append(variantIDs, v)
			}
		}
		var salesChannel, region *dal.Record
		var customerData cartsteps.CustomerData
		err = composer.Parallelize(
			func() error {
				var err error
				salesChannel, err = cartsteps.FindSalesChannelStep.Run(e, g(order, "sales_channel_id"))
				return err
			},
			func() error {
				var err error
				regionID, _ := g(order, "region_id").(string)
				region, err = cartsteps.FindOneOrAnyRegionStep.Run(e, regionID)
				return err
			},
			func() error {
				var err error
				// upstream passes { customer, email }: the step only reads customerId
				customerData, err = cartsteps.FindOrCreateCustomerStep.Run(e, cartsteps.FindOrCreateCustomerInput{Email: g(order, "email")})
				return err
			},
		)
		if err != nil {
			return nil, err
		}
		hookIn := dal.NewRecord()
		hookIn.Set("order", order)
		hookIn.Set("variantIds", variantIDs)
		hookIn.Set("region", region)
		hookIn.Set("customerData", customerData)
		if err := composer.CreateHook(e, "setPricingContext", hookIn); err != nil {
			return nil, err
		}
		vi, err := cartworkflows.GetVariantsAndItemsWithPrices.RunAsStep(e, cartworkflows.VariantsAndItemsInput{
			Cart:          order,
			Items:         in.Items,
			VariantIDs:    variantIDs,
			VariantFields: cartutils.Deduplicate(cartutils.ProductVariantsFields, cartutils.RequiredVariantFieldsForInventoryConfirmation),
		})
		if err != nil {
			return nil, err
		}
		channelID, _ := g(salesChannel, "id").(string)
		if _, err := cartworkflows.ConfirmVariantInventoryWorkflow.RunAsStep(e, cartutils.ConfirmVariantInventoryInput{SalesChannelID: channelID, Variants: vi.Variants, Items: in.Items}); err != nil {
			return nil, err
		}
		items := make([]*dal.Record, 0, len(vi.LineItems))
		for _, li := range vi.LineItems {
			items = append(items, rec(g(li, "data")))
		}
		translatedItems, err := commonsteps.GetTranslatedLineItemsStep.Run(e, commonsteps.GetTranslatedLineItemsStepInput{Items: items, Variants: vi.Variants, Locale: str(g(order, "locale"))})
		if err != nil {
			return nil, err
		}
		return ordersteps.CreateOrderLineItemsStep.Run(e, translatedItems)
	})

func queryClaim(e *composer.Execution, name string, id any, fields []string) (*dal.Record, error) {
	return remoteOne(e, name, "order_claim", idVars(id), fields, true)
}

func claimChangeValidationStep(name string) *composer.Step[changeValidationInput, any] {
	return composer.CreateStep(name,
		func(ctx composer.StepExecutionContext, in changeValidationInput) (composer.StepResponse[any], error) {
			if err := throwIfIsCancelled(in.Order, "Order"); err != nil {
				return composer.StepResponse[any]{}, err
			}
			if err := throwIfIsCancelled(in.OrderReturn, "Claim"); err != nil {
				return composer.StepResponse[any]{}, err
			}
			if err := ThrowIfOrderChangeIsNotActive(in.OrderChange); err != nil {
				return composer.StepResponse[any]{}, err
			}
			return composer.NewStepResponse[any](nil), nil
		}, nil)
}

var beginClaimOrderValidationStep = composer.CreateStep("begin-claim-order-validation",
	func(ctx composer.StepExecutionContext, order *dal.Record) (composer.StepResponse[any], error) {
		if err := throwIfIsCancelled(order, "Order"); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// BeginClaimOrderWorkflow mirrors beginClaimOrderWorkflow.
var BeginClaimOrderWorkflow = composer.CreateWorkflow("begin-claim-order",
	func(e *composer.Execution, in *dal.Record) (*dal.Record, error) {
		order, err := remoteOne(e, "", "orders", idVars(g(in, "order_id")), []string{"id", "status"}, true)
		if err != nil {
			return nil, err
		}
		if _, err := beginClaimOrderValidationStep.Run(e, order); err != nil {
			return nil, err
		}
		c := dal.NewRecord()
		c.Set("type", g(in, "type"))
		c.Set("order_id", g(in, "order_id"))
		setIfPresent(c, in, "metadata", "created_by")
		created, err := ordersteps.CreateOrderClaimsStep.Run(e, []*dal.Record{c})
		if err != nil {
			return nil, err
		}
		ch := dal.NewRecord()
		ch.Set("change_type", "claim")
		ch.Set("order_id", g(in, "order_id"))
		ch.Set("claim_id", g(created[0], "id"))
		setIfPresent(ch, in, "created_by", "description", "internal_note")
		return ordersteps.CreateOrderChangeStep.Run(e, ch)
	})

func claimActiveChange(e *composer.Execution, claim *dal.Record, fields []string) (*dal.Record, error) {
	return activeOrderChangeFor(e, g(claim, "order_id"), "claim_id", g(claim, "id"), fields)
}

func computeAdjustments(e *composer.Execution, order, orderChange *dal.Record) error {
	_, err := ComputeAdjustmentsForPreviewWorkflow.RunAsStep(e, ComputeAdjustmentsForPreviewInput{Order: withPromotions(order), OrderChange: orderChange})
	return err
}

func refreshClaimShipping(e *composer.Execution, orderChange, claim *dal.Record) error {
	args := dal.NewRecord()
	args.Set("order_change_id", g(orderChange, "id"))
	args.Set("claim_id", g(claim, "id"))
	args.Set("order_id", g(claim, "order_id"))
	_, err := RefreshClaimShippingWorkflow.RunAsStep(e, args)
	return err
}

var orderClaimItemValidationStep = claimChangeValidationStep("order-claim-item-validation")

// OrderClaimItemsInput mirrors OrderClaimItemWorkflowInput and friends.
type OrderClaimItemsInput struct {
	ClaimID    any
	Items      []*dal.Record
	LocationID any
}

// OrderClaimItemWorkflow mirrors orderClaimItemWorkflow.
var OrderClaimItemWorkflow = composer.CreateWorkflow("claim-item",
	func(e *composer.Execution, in OrderClaimItemsInput) (*dal.Record, error) {
		claim, err := queryClaim(e, "claim-query", in.ClaimID, []string{"id", "order_id", "canceled_at"})
		if err != nil {
			return nil, err
		}
		order, err := queryOrderOne(e, g(claim, "order_id"), append(append([]string{}, FieldsToComputeAdjustmentsForPreview...), "status", "canceled_at"))
		if err != nil {
			return nil, err
		}
		orderChange, err := claimActiveChange(e, claim, []string{"id", "status", "version", "claim_id", "carry_over_promotions"})
		if err != nil {
			return nil, err
		}
		if _, err := orderClaimItemValidationStep.Run(e, changeValidationInput{order, claim, orderChange}); err != nil {
			return nil, err
		}
		actions := make([]*dal.Record, 0, len(in.Items))
		for _, item := range in.Items {
			a := dal.NewRecord()
			a.Set("order_change_id", g(orderChange, "id"))
			a.Set("order_id", g(order, "id"))
			a.Set("claim_id", g(claim, "id"))
			a.Set("version", g(orderChange, "version"))
			a.Set("action", "WRITE_OFF_ITEM")
			setIfPresent(a, item, "internal_note")
			a.Set("reference", "order_claim")
			a.Set("reference_id", g(claim, "id"))
			d := dal.NewRecord()
			d.Set("reference_id", g(item, "id"))
			setIfPresent(d, item, "reason")
			d.Set("quantity", g(item, "quantity"))
			setIfPresent(d, item, "metadata")
			a.Set("details", d)
			actions = append(actions, a)
		}
		if _, err := CreateOrderChangeActionsWorkflow.RunAsStep(e, actions); err != nil {
			return nil, err
		}
		if err := computeAdjustments(e, order, orderChange); err != nil {
			return nil, err
		}
		return ordersteps.PreviewOrderChangeStep.Run(e, fmt.Sprint(g(claim, "order_id")))
	})

type claimReturnValidationInput struct {
	Order       *dal.Record
	OrderReturn *dal.Record
	OrderClaim  *dal.Record
	OrderChange *dal.Record
	Items       []*dal.Record
}

func throwIfManagedItemsNotStockedAtReturnLocation(order, orderReturn *dal.Record, inputItems []*dal.Record) error {
	locationID := g(orderReturn, "location_id")
	if !truthy(locationID) {
		return nil
	}
	wanted := map[any]bool{}
	for _, i := range inputItems {
		wanted[g(i, "id")] = true
	}
	var invalid []string
	for _, oi := range recs(g(order, "items")) {
		if !wanted[g(oi, "id")] {
			continue
		}
		variant := rec(g(oi, "variant"))
		if !truthy(g(variant, "manage_inventory")) {
			continue
		}
		found := false
		cartutils.DeepFlatMap(oi, "variant.inventory_items.inventory.location_levels", func(c map[string]*dal.Record) {
			if g(c["location_levels"], "location_id") == locationID {
				found = true
			}
		})
		if !found {
			invalid = append(invalid, fmt.Sprint(g(oi, "id")))
		}
	}
	if len(invalid) > 0 {
		return invalidData(fmt.Sprintf("Cannot request item return at location %v for managed inventory items: %s", locationID, strings.Join(invalid, ", ")))
	}
	return nil
}

var orderClaimRequestItemReturnValidationStep = composer.CreateStep("order-claim-request-item-return-validation",
	func(ctx composer.StepExecutionContext, in claimReturnValidationInput) (composer.StepResponse[any], error) {
		fail := func(err error) (composer.StepResponse[any], error) { return composer.StepResponse[any]{}, err }
		if err := throwIfIsCancelled(in.Order, "Order"); err != nil {
			return fail(err)
		}
		if err := throwIfIsCancelled(in.OrderClaim, "Claim"); err != nil {
			return fail(err)
		}
		if err := throwIfIsCancelled(in.OrderReturn, "Return"); err != nil {
			return fail(err)
		}
		if err := ThrowIfOrderChangeIsNotActive(in.OrderChange); err != nil {
			return fail(err)
		}
		if err := throwIfItemsDoesNotExistsInOrder(in.Order, in.Items); err != nil {
			return fail(err)
		}
		if err := throwIfManagedItemsNotStockedAtReturnLocation(in.Order, in.OrderReturn, in.Items); err != nil {
			return fail(err)
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// OrderClaimRequestItemReturnWorkflow mirrors orderClaimRequestItemReturnWorkflow.
var OrderClaimRequestItemReturnWorkflow = composer.CreateWorkflow("claim-request-item-return",
	func(e *composer.Execution, in OrderClaimItemsInput) (*dal.Record, error) {
		claim, err := queryClaim(e, "claim-query", in.ClaimID, []string{"id", "order_id", "return_id", "canceled_at"})
		if err != nil {
			return nil, err
		}
		var existingReturn *dal.Record
		if truthy(g(claim, "return_id")) {
			if existingReturn, err = remoteOne(e, "return-query", "return", idVars(g(claim, "return_id")), []string{"id", "status", "order_id", "location_id", "canceled_at"}, true); err != nil {
				return nil, err
			}
		}
		order, err := queryOrderOne(e, g(claim, "order_id"), append(append([]string{}, FieldsToComputeAdjustmentsForPreview...), "status",
			"items.variant.manage_inventory", "items.variant.inventory_items.inventory_item_id", "items.variant.inventory_items.inventory.location_levels.location_id"))
		if err != nil {
			return nil, err
		}
		orderChange, err := claimActiveChange(e, claim, []string{"id", "status", "version", "claim_id", "canceled_at", "confirmed_at", "declined_at", "carry_over_promotions"})
		if err != nil {
			return nil, err
		}
		locationID := in.LocationID
		if !truthy(locationID) {
			locationID = nil
			var item *dal.Record
			if len(in.Items) > 0 {
				for _, oi := range recs(g(order, "items")) {
					if g(oi, "id") == g(in.Items[0], "id") {
						item = oi
						break
					}
				}
			}
			if item != nil {
				cartutils.DeepFlatMap(item, "variant.inventory_items.inventory.location_levels", func(c map[string]*dal.Record) {
					if locationID == nil {
						if v := g(c["location_levels"], "location_id"); v != nil {
							locationID = v
						}
					}
				})
			}
		}
		orderReturn := existingReturn
		var createdReturn []*dal.Record
		if !truthy(g(claim, "return_id")) {
			r := dal.NewRecord()
			r.Set("order_id", g(claim, "order_id"))
			r.Set("claim_id", g(claim, "id"))
			if locationID != nil {
				r.Set("location_id", locationID)
			}
			if createdReturn, err = ordersteps.CreateReturnsStep.Run(e, []*dal.Record{r}); err != nil {
				return nil, err
			}
			if orderReturn == nil && len(createdReturn) > 0 {
				orderReturn = createdReturn[0]
			}
		}
		if len(createdReturn) > 0 {
			u := dal.NewRecord()
			u.Set("id", g(orderChange, "id"))
			u.Set("return_id", g(createdReturn[0], "id"))
			if _, err := ordersteps.UpdateOrderChangesStep.Run(e, []*dal.Record{u}); err != nil {
				return nil, err
			}
		}
		if _, err := orderClaimRequestItemReturnValidationStep.Run(e, claimReturnValidationInput{order, orderReturn, claim, orderChange, in.Items}); err != nil {
			return nil, err
		}
		if !truthy(g(claim, "return_id")) {
			u := dal.NewRecord()
			u.Set("id", g(claim, "id"))
			var rid any
			if len(createdReturn) > 0 {
				rid = g(createdReturn[0], "id")
			}
			u.Set("return_id", rid)
			if _, err := ordersteps.UpdateOrderClaimsStep.Run(e, []*dal.Record{u}); err != nil {
				return nil, err
			}
		}
		actions := make([]*dal.Record, 0, len(in.Items))
		for _, item := range in.Items {
			a := dal.NewRecord()
			a.Set("order_change_id", g(orderChange, "id"))
			a.Set("order_id", g(order, "id"))
			a.Set("claim_id", g(claim, "id"))
			a.Set("return_id", g(orderReturn, "id"))
			a.Set("version", g(orderChange, "version"))
			a.Set("action", "RETURN_ITEM")
			setIfPresent(a, item, "internal_note")
			a.Set("reference", "return")
			a.Set("reference_id", g(orderReturn, "id"))
			d := dal.NewRecord()
			d.Set("reference_id", g(item, "id"))
			d.Set("quantity", g(item, "quantity"))
			setIfPresent(d, item, "reason_id", "metadata")
			a.Set("details", d)
			actions = append(actions, a)
		}
		if _, err := CreateOrderChangeActionsWorkflow.RunAsStep(e, actions); err != nil {
			return nil, err
		}
		if err := computeAdjustments(e, order, orderChange); err != nil {
			return nil, err
		}
		if err := refreshClaimShipping(e, orderChange, claim); err != nil {
			return nil, err
		}
		return ordersteps.PreviewOrderChangeStep.Run(e, fmt.Sprint(g(claim, "order_id")))
	})

var orderClaimAddNewItemValidationStep = claimChangeValidationStep("order-claim-add-new-item-validation")

// OrderClaimAddNewItemWorkflow mirrors orderClaimAddNewItemWorkflow.
var OrderClaimAddNewItemWorkflow = composer.CreateWorkflow("claim-add-new-item",
	func(e *composer.Execution, in OrderClaimItemsInput) (*dal.Record, error) {
		claim, err := queryClaim(e, "claim-query", in.ClaimID, []string{"id", "order_id", "canceled_at"})
		if err != nil {
			return nil, err
		}
		order, err := queryOrderOne(e, g(claim, "order_id"), append(append([]string{}, FieldsToComputeAdjustmentsForPreview...), "status", "canceled_at"))
		if err != nil {
			return nil, err
		}
		orderChange, err := claimActiveChange(e, claim, []string{"id", "status", "version", "claim_id", "carry_over_promotions"})
		if err != nil {
			return nil, err
		}
		if _, err := orderClaimAddNewItemValidationStep.Run(e, changeValidationInput{order, claim, orderChange}); err != nil {
			return nil, err
		}
		lineItems, err := AddOrderLineItemsWorkflow.RunAsStep(e, AddOrderLineItemsInput{OrderID: g(order, "id"), Items: in.Items})
		if err != nil {
			return nil, err
		}
		itemIDs := []any{}
		for _, li := range lineItems {
			itemIDs = append(itemIDs, g(li, "id"))
		}
		if _, err := UpdateOrderTaxLinesWorkflow.RunAsStep(e, UpdateOrderTaxLinesInput{OrderID: g(order, "id"), ItemIDs: itemIDs}); err != nil {
			return nil, err
		}
		actions := make([]*dal.Record, 0, len(in.Items))
		for i, item := range in.Items {
			a := dal.NewRecord()
			a.Set("order_change_id", g(orderChange, "id"))
			a.Set("order_id", g(order, "id"))
			a.Set("claim_id", g(claim, "id"))
			a.Set("version", g(orderChange, "version"))
			a.Set("action", "ITEM_ADD")
			setIfPresent(a, item, "internal_note")
			a.Set("reference", "order_claim")
			a.Set("reference_id", g(claim, "id"))
			d := dal.NewRecord()
			d.Set("reference_id", g(lineItems[i], "id"))
			d.Set("quantity", g(item, "quantity"))
			price := g(item, "unit_price")
			if price == nil {
				price = g(lineItems[i], "unit_price")
			}
			d.Set("unit_price", price)
			setIfPresent(d, item, "metadata")
			a.Set("details", d)
			actions = append(actions, a)
		}
		if _, err := CreateOrderChangeActionsWorkflow.RunAsStep(e, actions); err != nil {
			return nil, err
		}
		if err := computeAdjustments(e, order, orderChange); err != nil {
			return nil, err
		}
		if err := refreshClaimShipping(e, orderChange, claim); err != nil {
			return nil, err
		}
		return ordersteps.PreviewOrderChangeStep.Run(e, fmt.Sprint(g(claim, "order_id")))
	})

// RefreshClaimShippingWorkflow mirrors refreshClaimShippingWorkflow.
var RefreshClaimShippingWorkflow = composer.CreateWorkflow("refresh-claim-shipping",
	func(e *composer.Execution, in *dal.Record) (any, error) {
		f := dal.NewRecord()
		f.Set("order_id", g(in, "order_id"))
		f.Set("claim_id", g(in, "claim_id"))
		f.Set("status", []any{"pending", "requested"})
		changes, err := queryList(e, "order-change-query", "order_change", f, []string{"id", "status", "order_id", "claim_id", "return_id", "actions.*"})
		if err != nil {
			return nil, err
		}
		var orderChange *dal.Record
		if len(changes) > 0 {
			orderChange = changes[0]
		}
		actions := recs(g(orderChange, "actions"))
		itemsOf := func(kind string) []any {
			out := []any{}
			for _, a := range actions {
				if g(a, "action") == kind {
					d := rec(g(a, "details"))
					it := dal.NewRecord()
					it.Set("id", g(d, "reference_id"))
					it.Set("quantity", g(d, "quantity"))
					out = append(out, it)
				}
			}
			return out
		}
		var inbound, outbound *dal.Record
		for _, a := range actions {
			if g(a, "action") != "SHIPPING_ADD" {
				continue
			}
			if truthy(g(a, "return_id")) && inbound == nil {
				inbound = a
			}
			if !truthy(g(a, "return_id")) && outbound == nil {
				outbound = a
			}
		}
		if inbound != nil {
			c := dal.NewRecord()
			c.Set("return_id", g(inbound, "return_id"))
			c.Set("return_items", itemsOf("RETURN_ITEM"))
			args := dal.NewRecord()
			args.Set("shipping_method_id", g(inbound, "reference_id"))
			args.Set("order_id", g(orderChange, "order_id"))
			args.Set("action_id", g(inbound, "id"))
			args.Set("context", c)
			if _, err := MaybeRefreshShippingMethodsWorkflow.RunAsStep(e, args); err != nil {
				return nil, err
			}
		}
		if outbound != nil {
			c := dal.NewRecord()
			c.Set("claim_id", g(outbound, "claim_id"))
			c.Set("claim_items", itemsOf("ITEM_ADD"))
			args := dal.NewRecord()
			args.Set("shipping_method_id", g(outbound, "reference_id"))
			args.Set("order_id", g(orderChange, "order_id"))
			args.Set("action_id", g(outbound, "id"))
			args.Set("context", c)
			if _, err := MaybeRefreshShippingMethodsWorkflow.RunAsStep(e, args); err != nil {
				return nil, err
			}
		}
		return nil, nil
	})

var createClaimShippingMethodValidationStep = claimChangeValidationStep("validate-create-claim-shipping-method")

// CreateClaimShippingMethodWorkflow mirrors createClaimShippingMethodWorkflow.
var CreateClaimShippingMethodWorkflow = composer.CreateWorkflow("create-claim-shipping-method",
	func(e *composer.Execution, in *dal.Record) (*dal.Record, error) {
		claim, err := remoteOne(e, "", "order_claim", idVars(g(in, "claim_id")), []string{"id", "status", "order_id", "canceled_at"}, true)
		if err != nil {
			return nil, err
		}
		order, err := queryOrderOne(e, g(claim, "order_id"), []string{"id", "status", "region_id", "currency_code", "canceled_at", "locale"})
		if err != nil {
			return nil, err
		}
		orderChange, err := claimActiveChange(e, claim, []string{"id", "status", "version", "actions.*"})
		if err != nil {
			return nil, err
		}
		isReturn := truthy(g(in, "return_id"))
		kind := "ITEM_ADD"
		if isReturn {
			kind = "RETURN_ITEM"
		}
		items := []any{}
		for _, a := range recs(g(orderChange, "actions")) {
			if g(a, "action") == kind {
				d := rec(g(a, "details"))
				it := dal.NewRecord()
				it.Set("id", g(d, "reference_id"))
				it.Set("quantity", g(d, "quantity"))
				items = append(items, it)
			}
		}
		c := dal.NewRecord()
		if isReturn {
			c.Set("return_id", g(in, "return_id"))
			c.Set("return_items", items)
		} else {
			c.Set("claim_id", g(in, "claim_id"))
			c.Set("claim_items", items)
		}
		option, err := FetchShippingOptionForOrderWorkflow.RunAsStep(e, FetchShippingOptionInput{
			OrderID: g(order, "id"), ShippingOptionID: g(in, "shipping_option_id"), CurrencyCode: g(order, "currency_code"), Context: c,
		})
		if err != nil {
			return nil, err
		}
		if option, err = TranslateShippingOption(e, option, g(order, "locale")); err != nil {
			return nil, err
		}
		if _, err := createClaimShippingMethodValidationStep.Run(e, changeValidationInput{order, claim, orderChange}); err != nil {
			return nil, err
		}
		customPrice := g(in, "custom_amount")
		method := PrepareShippingMethod("claim_id", option, orderChange, claim, in, customPrice)
		created, err := ordersteps.CreateOrderShippingMethodsStep.Run(e, []*dal.Record{method})
		if err != nil {
			return nil, err
		}
		var methodIDs []any
		for _, m := range created {
			methodIDs = append(methodIDs, g(m, "id"))
		}
		if _, err := UpdateOrderTaxLinesWorkflow.RunAsStep(e, UpdateOrderTaxLinesInput{OrderID: g(order, "id"), ShippingMethodIDs: methodIDs, IsReturn: isReturn}); err != nil {
			return nil, err
		}
		price := customPrice
		if price == nil {
			price = g(rec(g(option, "calculated_price")), "calculated_amount")
		}
		a := dal.NewRecord()
		a.Set("action", "SHIPPING_ADD")
		a.Set("reference", "order_shipping_method")
		a.Set("order_change_id", g(orderChange, "id"))
		a.Set("reference_id", g(created[0], "id"))
		a.Set("amount", price)
		a.Set("order_id", g(order, "id"))
		setIfPresent(a, in, "return_id")
		a.Set("claim_id", g(claim, "id"))
		if _, err := CreateOrderChangeActionsWorkflow.RunAsStep(e, []*dal.Record{a}); err != nil {
			return nil, err
		}
		return ordersteps.PreviewOrderChangeStep.Run(e, fmt.Sprint(g(order, "id")))
	})

var confirmClaimRequestValidationStep = claimChangeValidationStep("validate-confirm-claim-request")

var confirmIfClaimItemsArePresent = composer.CreateStep("confirm-if-items-are-present",
	func(ctx composer.StepExecutionContext, in struct {
		ClaimItems  []*dal.Record
		ReturnItems []*dal.Record
	}) (composer.StepResponse[any], error) {
		if len(in.ClaimItems) > 0 || len(in.ReturnItems) > 0 {
			return composer.NewStepResponse[any](nil), nil
		}
		return composer.StepResponse[any]{}, invalidData("Order claim request should have at least 1 item")
	}, nil)

// prepareClaimFulfillmentData mirrors prepareFulfillmentData of
// confirmClaimRequestWorkflow (upstream passes the order items).
func prepareClaimFulfillmentData(order *dal.Record, items []*dal.Record, option *dal.Record) *dal.Record {
	orderItems := map[any]*dal.Record{}
	for _, i := range recs(g(order, "items")) {
		orderItems[g(i, "id")] = i
	}
	fitems := []any{}
	for _, i := range items {
		oi := orderItems[g(i, "id")]
		if oi == nil {
			oi = rec(g(i, "item"))
		}
		fi := dal.NewRecord()
		if v, ok := i.Get("item_id"); ok {
			fi.Set("line_item_id", v)
		}
		fi.Set("quantity", g(i, "quantity"))
		title := g(oi, "variant_title")
		if title == nil {
			title = g(oi, "title")
		}
		fi.Set("title", title)
		sku := g(oi, "variant_sku")
		if !truthy(sku) {
			sku = ""
		}
		fi.Set("sku", sku)
		barcode := g(oi, "variant_barcode")
		if !truthy(barcode) {
			barcode = ""
		}
		fi.Set("barcode", barcode)
		fitems = append(fitems, fi)
	}
	location := rec(g(rec(g(rec(g(option, "service_zone")), "fulfillment_set")), "location"))
	addr := dal.NewRecord()
	if address := rec(g(location, "address")); address != nil {
		for _, k := range address.Keys() {
			if k == "id" {
				continue
			}
			v, _ := address.Get(k)
			addr.Set(k, v)
		}
	}
	in := dal.NewRecord()
	in.Set("location_id", g(location, "id"))
	in.Set("provider_id", g(option, "provider_id"))
	in.Set("shipping_option_id", g(option, "id"))
	in.Set("items", fitems)
	in.Set("delivery_address", addr)
	in.Set("order", order)
	return in
}

// ConfirmClaimRequestInput mirrors ConfirmClaimRequestWorkflowInput.
type ConfirmClaimRequestInput struct {
	ClaimID     any
	ConfirmedBy any
}

// ConfirmClaimRequestWorkflow mirrors confirmClaimRequestWorkflow.
var ConfirmClaimRequestWorkflow = composer.CreateWorkflow("confirm-claim-request",
	func(e *composer.Execution, in ConfirmClaimRequestInput) (*dal.Record, error) {
		claim, err := remoteOne(e, "", "order_claim", idVars(in.ClaimID), []string{"id", "status", "order_id", "canceled_at", "additional_items.id"}, true)
		if err != nil {
			return nil, err
		}
		order, err := queryOrderOne(e, g(claim, "order_id"), []string{"id", "version", "canceled_at", "items.*", "items.item.id", "shipping_address.*"})
		if err != nil {
			return nil, err
		}
		orderChange, err := claimActiveChange(e, claim, []string{"id", "status", "claim_id", "return_id", "actions.id", "actions.claim_id", "actions.return_id",
			"actions.action", "actions.details", "actions.reference", "actions.reference_id", "actions.internal_note"})
		if err != nil {
			return nil, err
		}
		if _, err := confirmClaimRequestValidationStep.Run(e, changeValidationInput{order, claim, orderChange}); err != nil {
			return nil, err
		}
		var claimActions, returnActions []*dal.Record
		for _, a := range recs(g(orderChange, "actions")) {
			switch g(a, "action") {
			case "RETURN_ITEM":
				returnActions = append(returnActions, a)
			case "ITEM_ADD", "WRITE_OFF_ITEM":
				claimActions = append(claimActions, a)
			}
		}
		var claimID, returnID any
		if len(claimActions) > 0 {
			claimID = g(claimActions[0], "claim_id")
		}
		if len(returnActions) > 0 {
			returnID = g(returnActions[0], "return_id")
		}
		preview, err := ordersteps.PreviewOrderChangeStep.Run(e, fmt.Sprint(g(order, "id")))
		if err != nil {
			return nil, err
		}
		var createdClaimItems, createdReturnItems []*dal.Record
		err = composer.Parallelize(
			func() error {
				var err error
				createdClaimItems, err = ordersteps.CreateOrderClaimItemsFromActionsStep.Run(e, ordersteps.CreateClaimItemsFromActionsInput{Changes: claimActions, ClaimID: claimID})
				return err
			},
			func() error {
				var err error
				createdReturnItems, err = ordersteps.CreateReturnItemsFromActionsStep.Run(e, ordersteps.CreateReturnItemsFromActionsInput{Changes: returnActions, ReturnID: returnID})
				return err
			},
		)
		if err != nil {
			return nil, err
		}
		if _, err := confirmIfClaimItemsArePresent.Run(e, struct {
			ClaimItems  []*dal.Record
			ReturnItems []*dal.Record
		}{createdClaimItems, createdReturnItems}); err != nil {
			return nil, err
		}
		var createdReturnID any
		if len(createdReturnItems) > 0 {
			createdReturnID = g(createdReturnItems[0], "return_id")
		}
		if _, err := ordersteps.ConfirmOrderChangesStep.Run(e, ordersteps.ConfirmOrderChangesInput{OrderID: g(order, "id"), Changes: []*dal.Record{orderChange}, ConfirmedBy: in.ConfirmedBy}); err != nil {
			return nil, err
		}
		if truthy(createdReturnID) {
			u := dal.NewRecord()
			u.Set("id", createdReturnID)
			u.Set("status", "requested")
			u.Set("requested_at", time.Now())
			if _, err := ordersteps.UpdateReturnsStep.Run(e, []*dal.Record{u}); err != nil {
				return nil, err
			}
		}
		var returnShippingMethod, claimShippingMethod *dal.Record
		for _, sm := range recs(g(preview, "shipping_methods")) {
			for _, a := range recs(g(sm, "actions")) {
				if g(a, "action") != "SHIPPING_ADD" {
					continue
				}
				if truthy(g(a, "return_id")) && g(a, "return_id") == createdReturnID {
					returnShippingMethod = sm
				} else if g(a, "claim_id") == g(claim, "id") {
					claimShippingMethod = sm
				}
			}
		}
		if claimShippingMethod != nil {
			full, err := queryClaim(e, "claim-query", in.ClaimID, []string{"id", "version", "canceled_at", "order.sales_channel_id", "additional_items.quantity",
				"additional_items.raw_quantity", "additional_items.item.id", "additional_items.item.variant.manage_inventory",
				"additional_items.item.variant.allow_backorder", "additional_items.item.variant.inventory_items.inventory_item_id",
				"additional_items.item.variant.inventory_items.required_quantity",
				"additional_items.item.variant.inventory_items.inventory.location_levels.stock_locations.id",
				"additional_items.item.variant.inventory_items.inventory.location_levels.stock_locations.name",
				"additional_items.item.variant.inventory_items.inventory.location_levels.stock_locations.sales_channels.id",
				"additional_items.item.variant.inventory_items.inventory.location_levels.stock_locations.sales_channels.name"})
			if err != nil {
				return nil, err
			}
			var variants, items []*dal.Record
			for _, ci := range recs(g(full, "additional_items")) {
				item := rec(g(ci, "item"))
				it := dal.NewRecord()
				it.Set("id", g(item, "id"))
				it.Set("variant_id", g(item, "variant_id"))
				q := g(ci, "raw_quantity")
				if q == nil {
					q = g(ci, "quantity")
				}
				it.Set("quantity", q)
				items = append(items, it)
				variants = append(variants, rec(g(item, "variant")))
			}
			channelID, _ := g(rec(g(full, "order")), "sales_channel_id").(string)
			inv, err := cartutils.PrepareConfirmInventoryInput(cartutils.ConfirmVariantInventoryInput{SalesChannelID: channelID, Variants: variants, Items: items})
			if err != nil {
				return nil, err
			}
			reservations, err := cartsteps.ReserveInventoryStep.Run(e, inv)
			if err != nil {
				return nil, err
			}
			events := []any{}
			for _, r := range reservations {
				ev := dal.NewRecord()
				ev.Set("id", g(r, "id"))
				ev.Set("order_id", g(order, "id"))
				events = append(events, ev)
			}
			if _, err := commonsteps.EmitEventStep.Config("emit-reservation-item-created", composer.StepOptions{}).Run(e, commonsteps.EmitEventStepInput{EventName: "reservation-item.created", Data: events}); err != nil {
				return nil, err
			}
		}
		if returnShippingMethod != nil && truthy(createdReturnID) {
			option, err := remoteOne(e, "claim-return-shipping-option", "shipping_options", idVars(g(returnShippingMethod, "shipping_option_id")),
				[]string{"id", "provider_id", "service_zone.fulfillment_set.location.id", "service_zone.fulfillment_set.location.address.*"}, false)
			if err != nil {
				return nil, err
			}
			data := prepareClaimFulfillmentData(order, recs(g(order, "items")), option)
			fulfillment, err := fulfillmentworkflows.CreateReturnFulfillmentWorkflow.RunAsStep(e, data)
			if err != nil {
				return nil, err
			}
			o := dal.NewRecord()
			o.Set("return_id", createdReturnID)
			f := dal.NewRecord()
			f.Set("fulfillment_id", g(fulfillment, "id"))
			link := dal.NewRecord()
			link.Set(utilsmodulessdk.ORDER, o)
			link.Set(utilsmodulessdk.FULFILLMENT, f)
			if _, err := commonsteps.CreateRemoteLinkStep.Config("claim-return-shipping-fulfillment-link", composer.StepOptions{}).Run(e, []*dal.Record{link}); err != nil {
				return nil, err
			}
		}
		pc := dal.NewRecord()
		pc.Set("order_id", g(order, "id"))
		if _, err := CreateOrUpdateOrderPaymentCollectionWorkflow.RunAsStep(e, pc); err != nil {
			return nil, err
		}
		ev := dal.NewRecord()
		ev.Set("order_id", g(order, "id"))
		ev.Set("claim_id", g(claim, "id"))
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "order.claim_created", Data: ev}); err != nil {
			return nil, err
		}
		return preview, nil
	})

var cancelBeginOrderClaimValidationStep = claimChangeValidationStep("validate-cancel-begin-order-claim")

// CancelBeginOrderClaimWorkflow mirrors cancelBeginOrderClaimWorkflow.
var CancelBeginOrderClaimWorkflow = composer.CreateWorkflow("cancel-begin-order-claim",
	func(e *composer.Execution, claimID any) (any, error) {
		claim, err := remoteOne(e, "", "order_claim", idVars(claimID), []string{"id", "status", "order_id", "return_id", "canceled_at"}, true)
		if err != nil {
			return nil, err
		}
		order, err := queryOrderOne(e, g(claim, "order_id"), []string{"id", "version", "canceled_at"})
		if err != nil {
			return nil, err
		}
		orderChange, err := claimActiveChange(e, claim, []string{"id", "status", "version", "actions.*"})
		if err != nil {
			return nil, err
		}
		if _, err := cancelBeginOrderClaimValidationStep.Run(e, changeValidationInput{order, claim, orderChange}); err != nil {
			return nil, err
		}
		shippingToRemove := []any{}
		for _, a := range recs(g(orderChange, "actions")) {
			if g(a, "action") == "SHIPPING_ADD" {
				shippingToRemove = append(shippingToRemove, g(a, "id"))
			}
		}
		return nil, composer.Parallelize(
			func() error { _, err := ordersteps.DeleteReturnsStep.Run(e, []any{g(claim, "return_id")}); return err },
			func() error { _, err := ordersteps.DeleteClaimsStep.Run(e, []any{g(claim, "id")}); return err },
			func() error {
				_, err := ordersteps.DeleteOrderChangesStep.Run(e, []any{g(orderChange, "id")})
				return err
			},
			func() error { _, err := ordersteps.DeleteOrderShippingMethodsStep.Run(e, shippingToRemove); return err },
		)
	})
