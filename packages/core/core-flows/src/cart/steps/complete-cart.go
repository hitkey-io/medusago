package steps

import (
	"context"
	"fmt"
	"strings"

	. "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/utils"
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	lockingsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/locking/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// ValidateCartItemsStep mirrors validateCartItemsStep.
var ValidateCartItemsStep = composer.CreateStep("validate-cart-items",
	func(ctx composer.StepExecutionContext, cart *dal.Record) (composer.StepResponse[any], error) {
		if len(Records(Get(cart, "items"))) == 0 {
			return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Cannot complete a cart with no items")
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// listRegionPaymentProviderIDs mirrors listRegionPaymentProviderIds.
func listRegionPaymentProviderIDs(ctx context.Context, c composer.Container, regionID any) (map[any]bool, error) {
	f := dal.NewRecord()
	f.Set("id", regionID)
	res, err := commonsteps.QueryOf(c).Graph(ctx, query.GraphInput{Entity: "region", Fields: []string{"payment_providers.id"}, Filters: f}, joiner.Options{})
	if err != nil {
		return nil, err
	}
	out := map[any]bool{}
	if len(res.Data) > 0 {
		region, _ := res.Data[0].(*dal.Record)
		for _, p := range Records(Get(region, "payment_providers")) {
			out[Get(p, "id")] = true
		}
	}
	return out, nil
}

var processablePaymentStatuses = []string{"pending", "requires_more", "authorized", "captured", "pending_authorization"}

// ValidateCartPaymentsStep mirrors validateCartPaymentsStep.
var ValidateCartPaymentsStep = composer.CreateStep("validate-cart-payments",
	func(ctx composer.StepExecutionContext, cart *dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		pc := Get(cart, "payment_collection")
		if totals.Gte(Coalesce(Get(cart, "credit_line_total"), 0), 0) && totals.Lte(Coalesce(Get(cart, "total"), 0), 0) {
			return composer.NewStepResponse([]*dal.Record{}), nil
		}
		if !IsPresent(pc) {
			return composer.StepResponse[[]*dal.Record]{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Payment collection has not been initiated for cart")
		}
		var toProcess []*dal.Record
		for _, ps := range Records(Get(pc.(*dal.Record), "payment_sessions")) {
			status, _ := Get(ps, "status").(string)
			for _, s := range processablePaymentStatuses {
				if s == status {
					toProcess = append(toProcess, ps)
					break
				}
			}
		}
		if len(toProcess) == 0 {
			return composer.StepResponse[[]*dal.Record]{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Payment sessions are required to complete cart")
		}
		if regionID := Get(cart, "region_id"); Truthy(regionID) {
			enabled, err := listRegionPaymentProviderIDs(ctx.Context, ctx.Container, regionID)
			if err != nil {
				return composer.StepResponse[[]*dal.Record]{}, err
			}
			var disabled []string
			seen := map[string]bool{}
			for _, ps := range toProcess {
				id := fmt.Sprint(Get(ps, "provider_id"))
				if !enabled[Get(ps, "provider_id")] && !seen[id] {
					seen[id] = true
					disabled = append(disabled, id)
				}
			}
			if len(disabled) > 0 {
				s, verb := "", "is"
				if len(disabled) > 1 {
					s, verb = "s", "are"
				}
				return composer.StepResponse[[]*dal.Record]{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA,
					fmt.Sprintf("Payment provider%s %s %s not enabled in the cart's region %v.", s, strings.Join(disabled, ", "), verb, regionID))
			}
		}
		return composer.NewStepResponse(toProcess), nil
	}, nil)

// CompensatePaymentIfNeededStep mirrors compensatePaymentIfNeededStep (the
// refund of a captured payment is not ported yet: system provider payments
// are never captured at completion).
var CompensatePaymentIfNeededStep = composer.CreateStep("compensate-payment-if-needed",
	func(ctx composer.StepExecutionContext, paymentSessionID any) (composer.StepResponse[any], error) {
		return composer.NewStepResponse[any](paymentSessionID, paymentSessionID), nil
	}, nil)

// ValidateShippingStepInput mirrors ValidateShippingInput.
type ValidateShippingStepInput struct {
	Cart            *dal.Record
	ShippingOptions []*dal.Record
}

// ValidateShippingStep mirrors validateShippingStep.
var ValidateShippingStep = composer.CreateStep("validate-shipping",
	func(ctx composer.StepExecutionContext, in ValidateShippingStepInput) (composer.StepResponse[any], error) {
		optionProfile := map[any]any{}
		for _, o := range in.ShippingOptions {
			optionProfile[Get(o, "id")] = Get(o, "shipping_profile_id")
		}
		var withShipping []*dal.Record
		for _, item := range Records(Get(in.Cart, "items")) {
			if Truthy(Get(item, "requires_shipping")) {
				withShipping = append(withShipping, item)
			}
		}
		methods := Records(Get(in.Cart, "shipping_methods"))
		if len(withShipping) > 0 && len(methods) == 0 {
			return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "No shipping method selected but the cart contains items that require shipping.")
		}
		available := map[any]bool{}
		for _, m := range methods {
			available[optionProfile[Get(m, "shipping_option_id")]] = true
		}
		for _, item := range withShipping {
			profile := Get(Rec(Rec(Rec(item, "variant"), "product"), "shipping_profile"), "id")
			if !available[profile] {
				return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "The cart items require shipping profiles that are not satisfied by the current shipping methods")
			}
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

type reservationService interface {
	CreateReservationItems(ctx context.Context, input []*dal.Record) ([]*dal.Record, error)
	DeleteReservationItems(ctx context.Context, ids []any) error
}

// reservationsCompensation is the compensation input { reservations,
// inventoryItemIds } of the reservation steps.
func reservationsCompensation(reservationIDs []any, inventoryItemIDs []any) *dal.Record {
	r := dal.NewRecord()
	r.Set("reservations", reservationIDs)
	r.Set("inventoryItemIds", inventoryItemIDs)
	return r
}

// ReserveInventoryStep mirrors reserveInventoryStep: the reservations are
// created (and deleted on compensation) while the inventory items are
// locked.
var ReserveInventoryStep = composer.CreateStep("reserve-inventory-step",
	func(ctx composer.StepExecutionContext, items []ConfirmInventoryItem) (composer.StepResponse[[]*dal.Record], error) {
		if len(items) == 0 {
			return composer.NewStepResponse([]*dal.Record{}, any(reservationsCompensation([]any{}, []any{}))), nil
		}
		svc := ctx.Container.Resolve(utilsmodulessdk.INVENTORY).(reservationService)
		inventoryItemIDs := make([]any, len(items))
		data := make([]*dal.Record, len(items))
		for i, it := range items {
			inventoryItemIDs[i] = it.InventoryItemID
			r := dal.NewRecord()
			r.Set("line_item_id", it.ID)
			r.Set("inventory_item_id", it.InventoryItemID)
			r.Set("quantity", totals.Mult(it.RequiredQuantity, it.Quantity))
			r.Set("allow_backorder", it.AllowBackorder)
			var loc any
			if len(it.LocationIDs) > 0 {
				loc = it.LocationIDs[0]
			}
			r.Set("location_id", loc)
			data[i] = r
		}
		var created []*dal.Record
		err := lockingsteps.Execute(ctx.Context, ctx.Container, lockingsteps.UniqueKeys(inventoryItemIDs), func(lctx context.Context) error {
			var err error
			created, err = svc.CreateReservationItems(lctx, data)
			return err
		})
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(created, any(reservationsCompensation(commonsteps.RecordIDs(created), inventoryItemIDs))), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		r, _ := data.(*dal.Record)
		if r == nil {
			return nil
		}
		raw, _ := r.Get("reservations")
		ids, _ := raw.([]any)
		if len(ids) == 0 {
			return nil
		}
		rawItems, _ := r.Get("inventoryItemIds")
		itemIDs, _ := rawItems.([]any)
		return lockingsteps.Execute(ctx.Context, ctx.Container, lockingsteps.UniqueKeys(itemIDs), func(lctx context.Context) error {
			return ctx.Container.Resolve(utilsmodulessdk.INVENTORY).(reservationService).DeleteReservationItems(lctx, ids)
		})
	})
