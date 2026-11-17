package workflows

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/steps"
	. "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// RefreshCartPromotionAdjustmentsInput mirrors RefreshCartPromotionAdjustmentsWorkflowInput.
type RefreshCartPromotionAdjustmentsInput struct {
	CartID                     string
	Cart                       *dal.Record
	PromoCodes                 []string
	AdditionalPromotionContext *dal.Record
}

// RefreshCartPromotionAdjustmentsWorkflow mirrors refreshCartPromotionAdjustmentsWorkflow.
var RefreshCartPromotionAdjustmentsWorkflow = composer.CreateWorkflow("refresh-cart-promotion-adjustments",
	func(e *composer.Execution, in RefreshCartPromotionAdjustmentsInput) (*dal.Record, error) {
		cart := in.Cart
		fetched, err := composer.When(e, "should-fetch-cart", cart == nil, func() (*dal.Record, error) {
			return fetchCart(e, "fetch-cart", in.CartID, CartFieldsForRefreshSteps, false)
		})
		if err != nil {
			return nil, err
		}
		if cart == nil {
			cart = fetched
		}
		actions, err := steps.GetActionsToComputeFromPromotionsStep.Run(e, steps.ActionsInput{ComputeActionContext: cart, PromotionCodesToApply: in.PromoCodes, AdditionalPromotionContext: in.AdditionalPromotionContext})
		if err != nil {
			return nil, err
		}
		prepared, err := steps.PrepareAdjustmentsFromPromotionActionsStep.Run(e, actions)
		if err != nil {
			return nil, err
		}
		toAny := func(rs []*dal.Record) []any {
			out := make([]any, len(rs))
			for i, r := range rs {
				out[i] = r
			}
			return out
		}
		err = composer.Parallelize(
			func() error {
				_, err := steps.RemoveLineItemAdjustmentsStep.Run(e, prepared.LineItemAdjustmentIDsToRemove)
				return err
			},
			func() error {
				_, err := steps.RemoveShippingMethodAdjustmentsStep.Run(e, prepared.ShippingMethodAdjustmentIDsToRemove)
				return err
			},
			func() error {
				_, err := steps.CreateLineItemAdjustmentsStep.Run(e, toAny(prepared.LineItemAdjustmentsToCreate))
				return err
			},
			func() error {
				_, err := steps.CreateShippingMethodAdjustmentsStep.Run(e, toAny(prepared.ShippingMethodAdjustmentsToCreate))
				return err
			},
			func() error {
				_, err := steps.UpdateCartPromotionsStep.Run(e, steps.UpdateCartPromotionsInput{ID: in.CartID, PromoCodes: prepared.ComputedPromotionCodes, Action: steps.PromotionActionReplace})
				return err
			},
		)
		if err != nil {
			return nil, err
		}
		out := dal.NewRecord()
		skipped := make([]any, len(prepared.SkippedPromoCodes))
		for i, s := range prepared.SkippedPromoCodes {
			skipped[i] = s
		}
		out.Set("skipped_promo_codes", skipped)
		return out, nil
	})
