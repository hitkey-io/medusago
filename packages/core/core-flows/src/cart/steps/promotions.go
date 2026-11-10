package steps

import (
	"context"

	. "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/utils"
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	promotionservices "github.com/hitkey-io/medusago/packages/modules/promotion/src/services"
)

// Promotion actions (PromotionActions).
const (
	PromotionActionAdd     = "add"
	PromotionActionRemove  = "remove"
	PromotionActionReplace = "replace"
)

// PromotionCodesInput mirrors GetPromotionCodesToApplyStepInput.
type PromotionCodesInput struct {
	Cart       *dal.Record
	PromoCodes []string
	Action     string
}

// GetPromotionCodesToApply mirrors getPromotionCodesToApply.
var GetPromotionCodesToApply = composer.CreateStep("get-promotion-codes-to-apply",
	func(ctx composer.StepExecutionContext, in PromotionCodesInput) (composer.StepResponse[[]string], error) {
		action := in.Action
		if action == "" {
			action = PromotionActionAdd
		}
		var codes []string
		seen := map[string]bool{}
		add := func(c string) {
			if !seen[c] {
				seen[c] = true
				codes = append(codes, c)
			}
		}
		del := func(c string) {
			if seen[c] {
				delete(seen, c)
				for i, x := range codes {
					if x == c {
						codes = append(codes[:i], codes[i+1:]...)
						break
					}
				}
			}
		}
		lines := append(Records(Get(in.Cart, "items")), Records(Get(in.Cart, "shipping_methods"))...)
		for _, l := range lines {
			for _, a := range Records(Get(l, "adjustments")) {
				if c := Str(a, "code"); c != "" {
					add(c)
				}
			}
		}
		if action == PromotionActionRemove {
			for _, c := range in.PromoCodes {
				del(c)
			}
		}
		if action == PromotionActionReplace {
			codes, seen = nil, map[string]bool{}
		}
		if action == PromotionActionAdd || action == PromotionActionReplace {
			valid := map[string]bool{}
			if len(in.PromoCodes) > 0 {
				ids := make([]any, len(in.PromoCodes))
				for i, c := range in.PromoCodes {
					ids[i] = c
				}
				f := dal.NewRecord()
				f.Set("code", ids)
				rows, err := graph(ctx, "promotion", []string{"id", "code"}, f)
				if err != nil {
					return composer.StepResponse[[]string]{}, err
				}
				for _, r := range Records(rows) {
					valid[Str(r, "code")] = true
				}
			}
			for _, c := range in.PromoCodes {
				if !valid[c] {
					return composer.StepResponse[[]string]{}, invalid("The promotion code %s is invalid", c)
				}
				add(c)
			}
		}
		if codes == nil {
			codes = []string{}
		}
		return composer.NewStepResponse(codes), nil
	}, nil)

// ActionsInput mirrors GetActionsToComputeFromPromotionsStepInput.
type ActionsInput struct {
	ComputeActionContext       *dal.Record
	PromotionCodesToApply      []string
	Options                    promotionservices.ComputeActionOptions
	AdditionalPromotionContext *dal.Record
}

// GetActionsToComputeFromPromotionsStep mirrors getActionsToComputeFromPromotionsStep.
var GetActionsToComputeFromPromotionsStep = composer.CreateStep("get-actions-to-compute-from-promotions",
	func(ctx composer.StepExecutionContext, in ActionsInput) (composer.StepResponse[[]*dal.Record], error) {
		svc, _ := ctx.Container.Resolve(utilsmodulessdk.PROMOTION).(interface {
			ComputeActions(ctx context.Context, codes []string, applicationContext *dal.Record, options promotionservices.ComputeActionOptions) ([]*dal.Record, error)
		})
		merged := Copy(in.ComputeActionContext)
		if in.AdditionalPromotionContext != nil {
			for _, k := range in.AdditionalPromotionContext.Keys() {
				v, _ := in.AdditionalPromotionContext.Get(k)
				merged.Set(k, v)
			}
		}
		actions, err := svc.ComputeActions(ctx.Context, in.PromotionCodesToApply, merged, in.Options)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(actions), nil
	}, nil)

// PreparedAdjustments mirrors PrepareAdjustmentsFromPromotionActionsStepOutput.
type PreparedAdjustments struct {
	LineItemAdjustmentsToCreate         []*dal.Record
	LineItemAdjustmentIDsToRemove       []any
	ShippingMethodAdjustmentsToCreate   []*dal.Record
	ShippingMethodAdjustmentIDsToRemove []any
	ComputedPromotionCodes              []string
	SkippedPromoCodes                   []*dal.Record
}

// PrepareAdjustmentsFromPromotionActionsStep mirrors prepareAdjustmentsFromPromotionActionsStep.
var PrepareAdjustmentsFromPromotionActionsStep = composer.CreateStep("prepare-adjustments-from-promotion-actions",
	func(ctx composer.StepExecutionContext, actions []*dal.Record) (composer.StepResponse[PreparedAdjustments], error) {
		out := PreparedAdjustments{LineItemAdjustmentsToCreate: []*dal.Record{}, LineItemAdjustmentIDsToRemove: []any{}, ShippingMethodAdjustmentsToCreate: []*dal.Record{}, ShippingMethodAdjustmentIDsToRemove: []any{}, ComputedPromotionCodes: []string{}, SkippedPromoCodes: []*dal.Record{}}
		if len(actions) == 0 {
			return composer.NewStepResponse(out), nil
		}
		codes := make([]any, len(actions))
		for i, a := range actions {
			codes[i] = Get(a, "code")
		}
		f := dal.NewRecord()
		f.Set("code", codes)
		rows, err := graph(ctx, "promotion", []string{"id", "code"}, f)
		if err != nil {
			return composer.StepResponse[PreparedAdjustments]{}, err
		}
		promoIDs := map[any]any{}
		for _, r := range Records(rows) {
			promoIDs[Get(r, "code")] = Get(r, "id")
		}
		skipped := func(a *dal.Record, reason string) {
			s := dal.NewRecord()
			s.Set("code", Get(a, "code"))
			s.Set("reason", reason)
			out.SkippedPromoCodes = append(out.SkippedPromoCodes, s)
		}
		for _, a := range actions {
			switch Get(a, "action") {
			case "addItemAdjustment":
				r := dal.NewRecord()
				r.Set("code", Get(a, "code"))
				r.Set("amount", totals.JSONNumber(Get(a, "amount")))
				r.Set("is_tax_inclusive", Get(a, "is_tax_inclusive"))
				r.Set("item_id", Get(a, "item_id"))
				r.Set("promotion_id", promoIDs[Get(a, "code")])
				out.LineItemAdjustmentsToCreate = append(out.LineItemAdjustmentsToCreate, r)
			case "removeItemAdjustment":
				out.LineItemAdjustmentIDsToRemove = append(out.LineItemAdjustmentIDsToRemove, Get(a, "adjustment_id"))
			case "addShippingMethodAdjustment":
				r := dal.NewRecord()
				r.Set("code", Get(a, "code"))
				r.Set("amount", totals.JSONNumber(Get(a, "amount")))
				r.Set("shipping_method_id", Get(a, "shipping_method_id"))
				r.Set("promotion_id", promoIDs[Get(a, "code")])
				out.ShippingMethodAdjustmentsToCreate = append(out.ShippingMethodAdjustmentsToCreate, r)
			case "removeShippingMethodAdjustment":
				out.ShippingMethodAdjustmentIDsToRemove = append(out.ShippingMethodAdjustmentIDsToRemove, Get(a, "adjustment_id"))
			case "promotionLimitExceeded":
				skipped(a, "promotion_limit_exceeded")
			case "campaignBudgetExceeded":
				skipped(a, "campaign_budget_exceeded")
			}
		}
		for _, r := range append(append([]*dal.Record{}, out.LineItemAdjustmentsToCreate...), out.ShippingMethodAdjustmentsToCreate...) {
			out.ComputedPromotionCodes = append(out.ComputedPromotionCodes, Str(r, "code"))
		}
		return composer.NewStepResponse(out), nil
	}, nil)

func adjustmentStep(name, className string, remove bool) *composer.Step[[]any, any] {
	return composer.CreateStep(name,
		func(ctx composer.StepExecutionContext, data []any) (composer.StepResponse[any], error) {
			if len(data) == 0 {
				return composer.NewStepResponse[any](nil, any([]any{})), nil
			}
			svc := CartOf(ctx.Container).Internal(className)
			if remove {
				if _, err := svc.SoftDelete(ctx.Context, data); err != nil {
					return composer.StepResponse[any]{}, err
				}
				return composer.NewStepResponse[any](nil, any(data)), nil
			}
			var recs []*dal.Record
			for _, d := range data {
				if r, ok := d.(*dal.Record); ok {
					recs = append(recs, r)
				}
			}
			created, err := svc.Create(ctx.Context, recs)
			if err != nil {
				return composer.StepResponse[any]{}, err
			}
			return composer.NewStepResponse[any](nil, any(commonsteps.RecordIDs(created))), nil
		},
		func(ctx composer.StepExecutionContext, data any) error {
			ids, _ := data.([]any)
			if len(ids) == 0 {
				return nil
			}
			svc := CartOf(ctx.Container).Internal(className)
			var err error
			if remove {
				_, err = svc.Restore(ctx.Context, ids)
			} else {
				_, err = svc.SoftDelete(ctx.Context, ids)
			}
			return err
		})
}

// Adjustment steps.
var (
	RemoveLineItemAdjustmentsStep       = adjustmentStep("remove-line-item-adjustments", "LineItemAdjustment", true)
	RemoveShippingMethodAdjustmentsStep = adjustmentStep("remove-shipping-method-adjustments", "ShippingMethodAdjustment", true)
	CreateLineItemAdjustmentsStep       = adjustmentStep("create-line-item-adjustments", "LineItemAdjustment", false)
	CreateShippingMethodAdjustmentsStep = adjustmentStep("create-shipping-method-adjustments", "ShippingMethodAdjustment", false)
)

// UpdateCartPromotionsInput mirrors UpdateCartPromotionStepInput.
type UpdateCartPromotionsInput struct {
	ID         string
	PromoCodes []string
	Action     string
}

type promoRevert struct {
	created   []*dal.Record
	dismissed []*dal.Record
}

func promoLink(cartID, promotionID any) *dal.Record {
	c := dal.NewRecord()
	c.Set("cart_id", cartID)
	p := dal.NewRecord()
	p.Set("promotion_id", promotionID)
	l := dal.NewRecord()
	l.Set(utilsmodulessdk.CART, c)
	l.Set(utilsmodulessdk.PROMOTION, p)
	return l
}

// UpdateCartPromotionsStep mirrors updateCartPromotionsStep.
var UpdateCartPromotionsStep = composer.CreateStep("update-cart-promotions",
	func(ctx composer.StepExecutionContext, in UpdateCartPromotionsInput) (composer.StepResponse[any], error) {
		action := in.Action
		if action == "" {
			action = PromotionActionAdd
		}
		v := dal.NewRecord()
		v.Set("cart_id", []any{in.ID})
		existing, err := commonsteps.RemoteQuery(ctx, "cart_promotion", v, []string{"cart_id", "promotion_id"})
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		existingMap := map[any]bool{}
		for _, l := range Records(existing) {
			existingMap[Get(l, "promotion_id")] = true
		}
		var toCreate, toDismiss []*dal.Record
		if len(in.PromoCodes) > 0 {
			codes := make([]any, len(in.PromoCodes))
			for i, c := range in.PromoCodes {
				codes[i] = c
			}
			f := dal.NewRecord()
			f.Set("code", codes)
			rows, err := graph(ctx, "promotion", []string{"id", "code"}, f)
			if err != nil {
				return composer.StepResponse[any]{}, err
			}
			for _, p := range Records(rows) {
				link := promoLink(in.ID, Get(p, "id"))
				if action == PromotionActionAdd || action == PromotionActionReplace {
					toCreate = append(toCreate, link)
				}
				if action == PromotionActionRemove && existingMap[Get(p, "id")] {
					toDismiss = append(toDismiss, link)
				}
			}
		}
		if action == PromotionActionReplace {
			for _, l := range Records(existing) {
				toDismiss = append(toDismiss, promoLink(Get(l, "cart_id"), Get(l, "promotion_id")))
			}
		}
		link := commonsteps.LinkOf(ctx.Container)
		if len(toDismiss) > 0 {
			if _, err := link.Dismiss(ctx.Context, toDismiss); err != nil {
				return composer.StepResponse[any]{}, err
			}
		}
		if len(toCreate) > 0 {
			if _, err := link.Create(ctx.Context, toCreate); err != nil {
				return composer.StepResponse[any]{}, err
			}
		}
		return composer.NewStepResponse[any](nil, any(promoRevert{toCreate, toDismiss})), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		r, ok := data.(promoRevert)
		if !ok {
			return nil
		}
		link := commonsteps.LinkOf(ctx.Container)
		if len(r.dismissed) > 0 {
			if _, err := link.Create(ctx.Context, r.dismissed); err != nil {
				return err
			}
		}
		if len(r.created) > 0 {
			if _, err := link.Dismiss(ctx.Context, r.created); err != nil {
				return err
			}
		}
		return nil
	})
