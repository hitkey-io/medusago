package workflows

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/steps"
	. "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/utils"
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	taxsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/tax/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

var updateTaxLinesCartFields = []string{"id", "currency_code", "email", "locale", "region.id", "region.automatic_taxes", "items.id", "items.variant_id", "items.product_id", "items.product_title", "items.product_description", "items.product_subtitle", "items.product_type", "items.product_type_id", "items.product_collection", "items.product_handle", "items.variant_sku", "items.variant_barcode", "items.variant_title", "items.title", "items.quantity", "items.unit_price", "items.is_giftcard", "items.tax_lines.id", "items.tax_lines.description", "items.tax_lines.code", "items.tax_lines.rate", "items.tax_lines.provider_id", "shipping_methods.tax_lines.id", "shipping_methods.tax_lines.description", "shipping_methods.tax_lines.code", "shipping_methods.tax_lines.rate", "shipping_methods.tax_lines.provider_id", "shipping_methods.shipping_option_id", "shipping_methods.amount", "shipping_methods.name", "shipping_methods.id", "customer.id", "customer.email", "customer.metadata", "customer.groups.id", "shipping_address.id", "shipping_address.address_1", "shipping_address.address_2", "shipping_address.city", "shipping_address.postal_code", "shipping_address.country_code", "shipping_address.region_code", "shipping_address.province", "shipping_address.metadata"}

// UpdateTaxLinesInput mirrors UpdateTaxLinesWorkflowInput.
type UpdateTaxLinesInput struct {
	CartID              string
	Cart                *dal.Record
	ForceTaxCalculation bool
}

func fetchCart(e *composer.Execution, name, id string, fields []string, throwIfNotFound bool) (*dal.Record, error) {
	isList := false
	f := dal.NewRecord()
	f.Set("id", id)
	res, err := commonsteps.UseQueryGraphStep.Config(name, composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
		GraphInput: query.GraphInput{Entity: "cart", Fields: fields, Filters: f},
		Options:    joiner.Options{ThrowIfKeyNotFound: throwIfNotFound},
		IsList:     &isList,
	})
	if err != nil {
		return nil, err
	}
	cart, _ := res.Data.(*dal.Record)
	return cart, nil
}

// UpdateTaxLinesWorkflow mirrors updateTaxLinesWorkflow.
var UpdateTaxLinesWorkflow = composer.CreateWorkflow("update-tax-lines",
	func(e *composer.Execution, in UpdateTaxLinesInput) (any, error) {
		cart := in.Cart
		if cart == nil {
			var err error
			cart, err = fetchCart(e, "fetch-cart", in.CartID, updateTaxLinesCartFields, true)
			if err != nil {
				return nil, err
			}
		}
		if _, err := steps.ValidateCartStep.Run(e, cart); err != nil {
			return nil, err
		}
		hookIn := dal.NewRecord()
		hookIn.Set("cart", cart)
		hookIn.Set("items", Get(cart, "items"))
		hookIn.Set("shipping_methods", Get(cart, "shipping_methods"))
		if err := composer.CreateHook(e, "setTaxLineContext", hookIn); err != nil {
			return nil, err
		}
		lines, err := taxsteps.GetItemTaxLinesStep.Run(e, taxsteps.GetItemTaxLinesInput{
			OrderOrCart:         cart,
			Items:               Records(Get(cart, "items")),
			ShippingMethods:     Records(Get(cart, "shipping_methods")),
			ForceTaxCalculation: in.ForceTaxCalculation,
		})
		if err != nil {
			return nil, err
		}
		if _, err := steps.SetTaxLinesForItemsStep.Run(e, steps.SetTaxLinesInput{Cart: cart, ItemTaxLines: lines.LineItemTaxLines, ShippingTaxLines: lines.ShippingMethodsTaxLines}); err != nil {
			return nil, err
		}
		return nil, nil
	})

// UpdateCartPromotionsInput mirrors UpdateCartPromotionsWorkflowInput.
type UpdateCartPromotionsInput struct {
	CartID                        string
	Cart                          *dal.Record
	PromoCodes                    []string
	Action                        string
	ForceRefreshPaymentCollection bool
}

// UpdateCartPromotionsWorkflow mirrors updateCartPromotionsWorkflow.
var UpdateCartPromotionsWorkflow = composer.CreateWorkflow("update-cart-promotions",
	func(e *composer.Execution, in UpdateCartPromotionsInput) (*dal.Record, error) {
		cart := in.Cart
		if cart == nil {
			var err error
			cart, err = fetchCart(e, "fetch-cart", in.CartID, CartFieldsForRefreshSteps, false)
			if err != nil {
				return nil, err
			}
		}
		if _, err := steps.ValidateCartStep.Run(e, cart); err != nil {
			return nil, err
		}
		action := in.Action
		if action == "" {
			action = steps.PromotionActionAdd
		}
		codes := in.PromoCodes
		if codes == nil {
			codes = []string{}
		}
		toApply, err := steps.GetPromotionCodesToApply.Run(e, steps.PromotionCodesInput{Cart: cart, PromoCodes: codes, Action: action})
		if err != nil {
			return nil, err
		}
		actions, err := steps.GetActionsToComputeFromPromotionsStep.Run(e, steps.ActionsInput{ComputeActionContext: cart, PromotionCodesToApply: toApply})
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
				_, err := steps.UpdateCartPromotionsStep.Run(e, steps.UpdateCartPromotionsInput{ID: Str(cart, "id"), PromoCodes: prepared.ComputedPromotionCodes, Action: steps.PromotionActionReplace})
				return err
			},
		)
		if err != nil {
			return nil, err
		}
		if in.ForceRefreshPaymentCollection {
			if _, err := RefreshPaymentCollectionForCartWorkflow.RunAsStep(e, RefreshPaymentCollectionInput{Cart: cart}); err != nil {
				return nil, err
			}
		}
		out := dal.NewRecord()
		skipped := make([]any, len(prepared.SkippedPromoCodes))
		for i, s := range prepared.SkippedPromoCodes {
			skipped[i] = s
		}
		out.Set("skipped_promo_codes", skipped)
		return out, nil
	})

// RefreshPaymentCollectionInput mirrors RefreshPaymentCollectionForCartWorklowInput.
type RefreshPaymentCollectionInput struct {
	CartID string
	Cart   *dal.Record
}

// RefreshPaymentCollectionForCartWorkflow mirrors refreshPaymentCollectionForCartWorkflow.
var RefreshPaymentCollectionForCartWorkflow = composer.CreateWorkflow("refresh-payment-collection-for-cart",
	func(e *composer.Execution, in RefreshPaymentCollectionInput) (any, error) {
		shouldExecute := in.CartID != ""
		if in.Cart != nil {
			shouldExecute = Truthy(Get(in.Cart, "payment_collection"))
		}
		cartID := in.CartID
		if cartID == "" {
			cartID = Str(in.Cart, "id")
		}
		cart := in.Cart
		if shouldExecute {
			fetched, err := fetchCart(e, "fetch-cart", cartID, []string{"id", "region_id", "currency_code", "total", "raw_total", "payment_collection.id", "payment_collection.raw_amount", "payment_collection.amount", "payment_collection.currency_code", "payment_collection.payment_sessions.id"}, true)
			if err != nil {
				return nil, err
			}
			if fetched != nil {
				cart = fetched
			}
		}
		if _, err := steps.ValidateCartStep.Run(e, cart); err != nil {
			return nil, err
		}
		hookIn := dal.NewRecord()
		hookIn.Set("cart", cart)
		if err := composer.CreateHook(e, "validate", hookIn); err != nil {
			return nil, err
		}
		pc := Rec(cart, "payment_collection")
		var rawAmount any = -1
		if pc != nil && Get(pc, "raw_amount") != nil {
			rawAmount = Get(pc, "raw_amount")
		}
		update := shouldExecute
		if equalBN(rawAmount, Get(cart, "raw_total")) {
			update = Get(pc, "currency_code") != Get(cart, "currency_code")
		}
		if update {
			if err := refreshPaymentCollection(e, cart); err != nil {
				return nil, err
			}
		}
		return nil, nil
	})
