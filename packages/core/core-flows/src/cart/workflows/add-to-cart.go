package workflows

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/steps"
	. "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/utils"
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	taxsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/tax/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

var addToCartCartFields = append([]string{"completed_at", "locale"}, FieldsForPricingContext...)

var variantFields = Deduplicate(ProductVariantsFields, RequiredVariantFieldsForInventoryConfirmation)

// AddToCartInput mirrors AddToCartWorkflowInputDTO.
type AddToCartInput struct {
	CartID         string
	Items          []*dal.Record
	AdditionalData *dal.Record
}

// AddToCartWorkflow mirrors addToCartWorkflow.
var AddToCartWorkflow = composer.CreateWorkflow("add-to-cart",
	func(e *composer.Execution, in AddToCartInput) (any, error) {
		if err := acquireLock(e, in.CartID, 2, 10); err != nil {
			return nil, err
		}
		cart, err := fetchCart(e, "get-cart", in.CartID, addToCartCartFields, true)
		if err != nil {
			return nil, err
		}
		if _, err := steps.ValidateCartStep.Run(e, cart); err != nil {
			return nil, err
		}
		var variantIDs []any
		for _, i := range in.Items {
			if v := Str(i, "variant_id"); v != "" {
				variantIDs = append(variantIDs, v)
			}
		}
		var variants []*dal.Record
		var lineItems []*dal.Record
		if len(variantIDs) > 0 {
			quantities, err := steps.GetLineItemPricingQuantitiesStep.Run(e, steps.LineItemsInput{ID: in.CartID, Items: in.Items})
			if err != nil {
				return nil, err
			}
			toPrice := make([]*dal.Record, len(in.Items))
			for i, item := range in.Items {
				c := Copy(item)
				if i < len(quantities) && quantities[i] != nil {
					c.Set("quantity", quantities[i])
				}
				toPrice[i] = c
			}
			vi, err := GetVariantsAndItemsWithPrices.RunAsStep(e, VariantsAndItemsInput{Cart: cart, Items: toPrice, VariantIDs: variantIDs, VariantFields: variantFields})
			if err != nil {
				return nil, err
			}
			variants = vi.Variants
			for i, li := range vi.LineItems {
				d := Copy(Rec(li, "data"))
				if i < len(in.Items) && Get(in.Items[i], "quantity") != nil {
					d.Set("quantity", Get(in.Items[i], "quantity"))
				}
				lineItems = append(lineItems, d)
			}
		}
		if len(variants) == 0 && len(variantIDs) > 0 {
			f := dal.NewRecord()
			f.Set("id", variantIDs)
			res, err := commonsteps.UseQueryGraphStep.Config("fetch-variants", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{GraphInput: query.GraphInput{Entity: "variants", Fields: variantFields, Filters: f}})
			if err != nil {
				return nil, err
			}
			variants = Records(res.Data)
		}
		if len(lineItems) == 0 {
			for _, item := range in.Items {
				var variant *dal.Record
				for _, v := range variants {
					if Get(v, "id") == Get(item, "variant_id") {
						variant = v
						break
					}
				}
				cp := Rec(variant, "calculated_price")
				inclusive := Get(item, "is_tax_inclusive")
				if inclusive == nil {
					inclusive = Get(cp, "is_calculated_price_tax_inclusive")
				}
				pin := PrepareLineItemDataInput{Item: item, Variant: variant, CartID: in.CartID, UnitPrice: Get(item, "unit_price"), IsTaxInclusive: Truthy(inclusive), IsCustomPrice: Has(item, "unit_price")}
				if variant != nil && pin.UnitPrice == nil {
					pin.UnitPrice = Get(cp, "calculated_amount")
				}
				li, err := PrepareLineItemData(pin)
				if err != nil {
					return nil, err
				}
				lineItems = append(lineItems, li)
			}
		}
		if _, err := steps.ValidateLineItemPricesStep.Run(e, lineItems); err != nil {
			return nil, err
		}
		actions, err := steps.GetLineItemActionsStep.Run(e, steps.LineItemsInput{ID: Str(cart, "id"), Items: lineItems})
		if err != nil {
			return nil, err
		}
		var toConfirm []*dal.Record
		for _, i := range append(append([]*dal.Record{}, actions.ItemsToUpdate...), actions.ItemsToCreate...) {
			v := Get(i, "variant_id")
			if d := Rec(i, "data"); d != nil {
				v = Get(d, "variant_id")
			}
			if v != nil {
				toConfirm = append(toConfirm, i)
			}
		}
		if _, err := ConfirmVariantInventoryWorkflow.RunAsStep(e, ConfirmVariantInventoryInput{SalesChannelID: Str(cart, "sales_channel_id"), Variants: variants, Items: in.Items, ItemsToUpdate: toConfirm}); err != nil {
			return nil, err
		}
		variantsByID := map[any]*dal.Record{}
		for _, v := range variants {
			variantsByID[Get(v, "id")] = v
		}
		var itemsToCreateVariants []*dal.Record
		for _, i := range actions.ItemsToCreate {
			if v, ok := variantsByID[Get(i, "variant_id")]; ok && Get(i, "variant_id") != nil {
				itemsToCreateVariants = append(itemsToCreateVariants, v)
			}
		}
		translatedItemsToCreate, err := commonsteps.GetTranslatedLineItemsStep.Run(e, commonsteps.GetTranslatedLineItemsStepInput{Items: actions.ItemsToCreate, Variants: itemsToCreateVariants, Locale: Str(cart, "locale")})
		if err != nil {
			return nil, err
		}
		var created, updated []*dal.Record
		err = composer.Parallelize(
			func() error {
				var err error
				created, err = steps.CreateLineItemsStep.Run(e, steps.LineItemsInput{ID: Str(cart, "id"), Items: translatedItemsToCreate})
				return err
			},
			func() error {
				var err error
				updated, err = steps.UpdateLineItemsStep.Run(e, steps.LineItemsInput{ID: Str(cart, "id"), Items: actions.ItemsToUpdate})
				return err
			},
		)
		if err != nil {
			return nil, err
		}
		all := append(append([]*dal.Record{}, created...), updated...)
		if _, err := RefreshCartItemsWorkflow.RunAsStep(e, RefreshCartItemsInput{CartID: Str(cart, "id"), Items: all, AdditionalData: in.AdditionalData}); err != nil {
			return nil, err
		}
		ev := dal.NewRecord()
		ev.Set("id", Get(cart, "id"))
		if err := composer.Parallelize(
			func() error {
				_, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "cart.updated", Data: ev})
				return err
			},
			func() error { return releaseLock(e, Str(cart, "id")) },
		); err != nil {
			return nil, err
		}
		return nil, nil
	})

// RefreshCartItemsInput mirrors RefreshCartItemsWorkflowInput.
type RefreshCartItemsInput struct {
	CartID              string
	PromoCodes          []string
	PromoCodesDefined   bool
	ForceRefresh        bool
	Items               []*dal.Record
	ShippingMethods     []*dal.Record
	ForceTaxCalculation bool
	AdditionalData      *dal.Record
	Locale              string
}

// RefreshCartItemsWorkflow mirrors refreshCartItemsWorkflow.
var RefreshCartItemsWorkflow = composer.CreateWorkflow("refresh-cart-items",
	func(e *composer.Execution, in RefreshCartItemsInput) (*dal.Record, error) {
		if err := acquireLock(e, in.CartID, 2, 10); err != nil {
			return nil, err
		}
		if in.ForceRefresh {
			cart, err := fetchCart(e, "use-query-graph-step", in.CartID, CartFieldsForRefreshSteps, false)
			if err != nil {
				return nil, err
			}
			if _, err := steps.ValidateCartStep.Run(e, cart); err != nil {
				return nil, err
			}
			vi, err := GetVariantsAndItemsWithPrices.RunAsStep(e, VariantsAndItemsInput{Cart: cart})
			if err != nil {
				return nil, err
			}
			if _, err := steps.UpdateLineItemsStep.Run(e, steps.LineItemsInput{ID: Str(cart, "id"), Items: vi.LineItems}); err != nil {
				return nil, err
			}
		}
		refetched, err := fetchCart(e, "refetch-cart", in.CartID, CartFieldsForRefreshSteps, false)
		if err != nil {
			return nil, err
		}
		if _, err := RefreshCartShippingMethodsWorkflow.RunAsStep(e, RefreshShippingMethodsInput{Cart: refetched, AdditionalData: in.AdditionalData}); err != nil {
			return nil, err
		}
		if in.ForceRefresh {
			if _, err := UpdateTaxLinesWorkflow.RunAsStep(e, UpdateTaxLinesInput{CartID: in.CartID}); err != nil {
				return nil, err
			}
		}
		if !in.ForceRefresh && (len(in.Items) > 0 || len(in.ShippingMethods) > 0) {
			if _, err := UpsertTaxLinesWorkflow.RunAsStep(e, UpsertTaxLinesInput{Cart: refetched, Items: in.Items, ShippingMethods: in.ShippingMethods, ForceTaxCalculation: in.ForceTaxCalculation}); err != nil {
				return nil, err
			}
		}
		codes := in.PromoCodes
		if !in.PromoCodesDefined {
			codes = []string{}
			for _, p := range Records(Get(refetched, "promotions")) {
				if c := Str(p, "code"); c != "" {
					codes = append(codes, c)
				}
			}
		}
		if _, err := UpdateCartPromotionsWorkflow.RunAsStep(e, UpdateCartPromotionsInput{CartID: in.CartID, PromoCodes: codes, Action: steps.PromotionActionReplace}); err != nil {
			return nil, err
		}
		if in.Locale != "" {
			if _, err := steps.UpdateCartItemsTranslationsStep.Run(e, steps.UpdateCartItemsTranslationsStepInput{CartID: in.CartID, Locale: in.Locale, Items: Records(Get(refetched, "items"))}); err != nil {
				return nil, err
			}
		}
		if _, err := RefreshPaymentCollectionForCartWorkflow.RunAsStep(e, RefreshPaymentCollectionInput{Cart: refetched}); err != nil {
			return nil, err
		}
		if err := releaseLock(e, in.CartID); err != nil {
			return nil, err
		}
		return refetched, nil
	})

// RefreshShippingMethodsInput mirrors RefreshCartShippingMethodsWorkflowInput.
type RefreshShippingMethodsInput struct {
	CartID         string
	Cart           *dal.Record
	AdditionalData *dal.Record
}

// RefreshCartShippingMethodsWorkflow mirrors refreshCartShippingMethodsWorkflow.
var RefreshCartShippingMethodsWorkflow = composer.CreateWorkflow("refresh-cart-shipping-methods",
	func(e *composer.Execution, in RefreshShippingMethodsInput) (any, error) {
		shouldExecute := in.CartID != ""
		if in.Cart != nil {
			shouldExecute = len(Records(Get(in.Cart, "shipping_methods"))) > 0
		}
		cart := in.Cart
		if shouldExecute {
			id := in.CartID
			if id == "" {
				id = Str(in.Cart, "id")
			}
			fetched, err := fetchCart(e, "get-cart", id, []string{"id", "sales_channel_id", "currency_code", "region_id", "shipping_methods.*", "shipping_address.city", "shipping_address.country_code", "shipping_address.province", "shipping_methods.shipping_option_id", "shipping_methods.data", "total", "items.requires_shipping", "items.variant.product.shipping_profile.id"}, true)
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
		if err := acquireLock(e, Str(cart, "id"), 2, 10); err != nil {
			return nil, err
		}
		var input []*dal.Record
		for _, m := range Records(Get(cart, "shipping_methods")) {
			o := dal.NewRecord()
			o.Set("id", Get(m, "shipping_option_id"))
			if v, ok := m.Get("data"); ok {
				o.Set("data", v)
			}
			input = append(input, o)
		}
		if len(input) == 0 {
			return nil, nil
		}
		options, err := ListShippingOptionsForCartWithPricingWorkflow.RunAsStep(e, PricedOptionsInput{CartID: Str(cart, "id"), Options: input, AdditionalData: in.AdditionalData})
		if err != nil {
			return nil, err
		}
		methods := Records(Get(cart, "shipping_methods"))
		cleanup := len(methods) > 1
		required := map[any]bool{}
		for _, item := range Records(Get(cart, "items")) {
			if Truthy(Get(item, "requires_shipping")) {
				if p := Get(Rec(Rec(Rec(item, "variant"), "product"), "shipping_profile"), "id"); Truthy(p) {
					required[p] = true
				}
			}
		}
		var toRemove []any
		var toUpdate []*dal.Record
		for _, m := range methods {
			so := findOption(options, Get(m, "shipping_option_id"))
			price := Get(Rec(so, "calculated_price"), "calculated_amount")
			valid := so != nil && price != nil
			if valid && cleanup {
				if p := Get(so, "shipping_profile_id"); Truthy(p) && !required[p] {
					valid = false
				}
			}
			if !valid {
				toRemove = append(toRemove, Get(m, "id"))
				continue
			}
			u := dal.NewRecord()
			u.Set("id", Get(m, "id"))
			u.Set("shipping_option_id", Get(so, "id"))
			u.Set("name", Get(so, "name"))
			u.Set("amount", Get(Rec(so, "calculated_price"), "calculated_amount"))
			u.Set("is_tax_inclusive", Get(Rec(so, "calculated_price"), "is_calculated_price_tax_inclusive"))
			toUpdate = append(toUpdate, u)
		}
		err = composer.Parallelize(
			func() error { _, err := steps.RemoveShippingMethodFromCartStep.Run(e, toRemove); return err },
			func() error { _, err := steps.UpdateShippingMethodsStep.Run(e, toUpdate); return err },
		)
		if err != nil {
			return nil, err
		}
		// released only when the shipping methods were refreshed (inside the
		// when() block upstream)
		return nil, releaseLock(e, Str(cart, "id"))
	})

// UpsertTaxLinesInput mirrors UpsertTaxLinesWorkflowInput.
type UpsertTaxLinesInput struct {
	CartID              string
	Cart                *dal.Record
	Items               []*dal.Record
	ShippingMethods     []*dal.Record
	ForceTaxCalculation bool
}

var upsertTaxLinesCartFields = []string{"id", "locale", "currency_code", "email", "region.id", "region.automatic_taxes", "items.id", "items.variant_id", "items.product_id", "items.product.is_giftcard", "items.product_title", "items.product_description", "items.product_subtitle", "items.product_type", "items.product_type_id", "items.product_collection", "items.product_handle", "items.variant_sku", "items.variant_barcode", "items.variant_title", "items.title", "items.quantity", "items.unit_price", "items.tax_lines.id", "items.tax_lines.description", "items.tax_lines.code", "items.tax_lines.rate", "items.tax_lines.provider_id", "shipping_methods.tax_lines.id", "shipping_methods.tax_lines.description", "shipping_methods.tax_lines.code", "shipping_methods.tax_lines.rate", "shipping_methods.tax_lines.provider_id", "shipping_methods.shipping_option_id", "shipping_methods.amount", "customer.id", "customer.email", "customer.metadata", "customer.groups.id", "shipping_address.id", "shipping_address.address_1", "shipping_address.address_2", "shipping_address.city", "shipping_address.postal_code", "shipping_address.country_code", "shipping_address.region_code", "shipping_address.province", "shipping_address.metadata"}

// UpsertTaxLinesWorkflow mirrors upsertTaxLinesWorkflow.
var UpsertTaxLinesWorkflow = composer.CreateWorkflow("upsert-tax-lines",
	func(e *composer.Execution, in UpsertTaxLinesInput) (any, error) {
		cart := in.Cart
		if cart == nil {
			var err error
			cart, err = fetchCart(e, "fetch-cart", in.CartID, upsertTaxLinesCartFields, true)
			if err != nil {
				return nil, err
			}
		}
		if _, err := steps.ValidateCartStep.Run(e, cart); err != nil {
			return nil, err
		}
		h := dal.NewRecord()
		h.Set("cart", cart)
		h.Set("items", in.Items)
		h.Set("shipping_methods", in.ShippingMethods)
		if err := composer.CreateHook(e, "setTaxLineContext", h); err != nil {
			return nil, err
		}
		lines, err := taxsteps.GetItemTaxLinesStep.Run(e, taxsteps.GetItemTaxLinesInput{OrderOrCart: cart, Items: in.Items, ShippingMethods: in.ShippingMethods, ForceTaxCalculation: in.ForceTaxCalculation})
		if err != nil {
			return nil, err
		}
		translated, err := commonsteps.GetTranslatedTaxLinesStep.Run(e, commonsteps.GetTranslatedTaxLinesStepInput{ItemTaxLines: lines.LineItemTaxLines, ShippingTaxLines: lines.ShippingMethodsTaxLines, Locale: Str(cart, "locale")})
		if err != nil {
			return nil, err
		}
		_, err = steps.UpsertTaxLinesForItemsStep.Run(e, steps.SetTaxLinesInput{Cart: cart, ItemTaxLines: translated.ItemTaxLines, ShippingTaxLines: translated.ShippingTaxLines})
		return nil, err
	})
