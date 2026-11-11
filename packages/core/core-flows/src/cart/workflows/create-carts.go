package workflows

import (
	"fmt"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/steps"
	. "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/utils"
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

type prepareCartInput struct {
	input        *dal.Record
	region       *dal.Record
	customerData steps.CustomerData
	salesChannel *dal.Record
}

var prepareCartToCreateStep = composer.CreateStep("prepare-cart-to-create",
	func(ctx composer.StepExecutionContext, d prepareCartInput) (composer.StepResponse[*dal.Record], error) {
		if d.region == nil {
			return composer.StepResponse[*dal.Record]{}, common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, "No regions found")
		}
		code := Str(d.input, "currency_code")
		regionCode := Str(d.region, "currency_code")
		if code != "" && strings.ToLower(code) != strings.ToLower(regionCode) {
			return composer.StepResponse[*dal.Record]{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Currency code %s does not match the region's currency code %s", code, regionCode))
		}
		data := Copy(d.input)
		data.Set("currency_code", Coalesce(Get(d.input, "currency_code"), regionCode))
		data.Set("region_id", Get(d.region, "id"))
		if c := d.customerData.Customer; c != nil && Truthy(Get(c, "id")) {
			data.Set("customer_id", Get(c, "id"))
			data.Set("email", Coalesce(Get(d.input, "email"), Get(c, "email")))
		}
		data.Set("sales_channel_id", Get(d.salesChannel, "id"))
		countries := Records(Get(d.region, "countries"))
		if !Truthy(Get(d.input, "shipping_address")) && len(countries) == 1 {
			a := dal.NewRecord()
			a.Set("country_code", Get(countries[0], "iso_2"))
			data.Set("shipping_address", a)
		}
		return composer.NewStepResponse(data), nil
	}, nil)

// CreateCartWorkflow mirrors createCartWorkflow. The input is the request
// body plus customer_id.
var CreateCartWorkflow = composer.CreateWorkflow("create-cart",
	func(e *composer.Execution, input *dal.Record) (*dal.Record, error) {
		items := Records(Get(input, "items"))
		var variantIDs []any
		for _, i := range items {
			if v := Str(i, "variant_id"); v != "" {
				variantIDs = append(variantIDs, v)
			}
		}
		if variantIDs == nil {
			variantIDs = []any{}
		}
		var salesChannel, region *dal.Record
		var customerData steps.CustomerData
		var channelArg any
		if v, ok := input.Get("sales_channel_id"); ok {
			if v == nil {
				channelArg = ""
			} else {
				channelArg = v
			}
		}
		hasAccount := false
		err := composer.Parallelize(
			func() error {
				var err error
				salesChannel, err = steps.FindSalesChannelStep.Run(e, channelArg)
				return err
			},
			func() error {
				var err error
				region, err = steps.FindOneOrAnyRegionStep.Run(e, Str(input, "region_id"))
				return err
			},
			func() error {
				var err error
				customerData, err = steps.FindOrCreateCustomerStep.Run(e, steps.FindOrCreateCustomerInput{CustomerID: Get(input, "customer_id"), Email: Get(input, "email"), HasAccount: &hasAccount})
				return err
			},
		)
		if err != nil {
			return nil, err
		}
		if _, err := steps.ValidateSalesChannelStep.Run(e, salesChannel); err != nil {
			return nil, err
		}
		hookIn := dal.NewRecord()
		hookIn.Set("region", region)
		hookIn.Set("variantIds", variantIDs)
		hookIn.Set("salesChannel", salesChannel)
		hookIn.Set("additional_data", Get(input, "additional_data"))
		if err := composer.CreateHook(e, "setPricingContext", hookIn); err != nil {
			return nil, err
		}
		cartForPricing := dal.NewRecord()
		if v, ok := input.Get("currency_code"); ok && v != nil {
			cartForPricing.Set("currency_code", v)
		}
		if region != nil {
			cartForPricing.Set("region", region)
			cartForPricing.Set("region_id", Get(region, "id"))
		}
		if c := customerData.Customer; c != nil {
			cartForPricing.Set("customer_id", Get(c, "id"))
			cartForPricing.Set("customer", c)
		}
		vi, err := GetVariantsAndItemsWithPrices.RunAsStep(e, VariantsAndItemsInput{
			Cart: cartForPricing, Items: itemsOrEmpty(items), VariantIDs: variantIDs,
			VariantFields: Deduplicate(ProductVariantsFields, RequiredVariantFieldsForInventoryConfirmation),
		})
		if err != nil {
			return nil, err
		}
		if _, err := ConfirmVariantInventoryWorkflow.RunAsStep(e, ConfirmVariantInventoryInput{SalesChannelID: Str(salesChannel, "id"), Variants: vi.Variants, Items: items}); err != nil {
			return nil, err
		}
		cartInput, err := prepareCartToCreateStep.Run(e, prepareCartInput{input: input, region: region, customerData: customerData, salesChannel: salesChannel})
		if err != nil {
			return nil, err
		}
		itemsToCreate := make([]*dal.Record, 0, len(vi.LineItems))
		for _, li := range vi.LineItems {
			if d, ok := Get(li, "data").(*dal.Record); ok {
				itemsToCreate = append(itemsToCreate, d)
			}
		}
		translatedItems, err := commonsteps.GetTranslatedLineItemsStep.Run(e, commonsteps.GetTranslatedLineItemsStepInput{Items: itemsToCreate, Variants: vi.Variants, Locale: Str(input, "locale")})
		if err != nil {
			return nil, err
		}
		lineItems := make([]any, len(translatedItems))
		for i, li := range translatedItems {
			lineItems[i] = li
		}
		cartInput.Set("items", lineItems)
		vh := dal.NewRecord()
		vh.Set("input", cartInput)
		vh.Set("cart", cartInput)
		if err := composer.CreateHook(e, "validate", vh); err != nil {
			return nil, err
		}
		carts, err := steps.CreateCartsStep.Run(e, []*dal.Record{cartInput})
		if err != nil {
			return nil, err
		}
		cart := carts[0]
		if _, err := UpdateTaxLinesWorkflow.RunAsStep(e, UpdateTaxLinesInput{CartID: Str(cart, "id")}); err != nil {
			return nil, err
		}
		codes := []string{}
		for _, c := range asStrings(Get(input, "promo_codes")) {
			codes = append(codes, c)
		}
		if _, err := UpdateCartPromotionsWorkflow.RunAsStep(e, UpdateCartPromotionsInput{CartID: Str(cart, "id"), PromoCodes: codes}); err != nil {
			return nil, err
		}
		err = composer.Parallelize(
			func() error {
				ev := dal.NewRecord()
				ev.Set("id", Get(cart, "id"))
				_, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "cart.created", Data: ev})
				return err
			},
			func() error {
				_, err := RefreshPaymentCollectionForCartWorkflow.RunAsStep(e, RefreshPaymentCollectionInput{Cart: cart})
				return err
			},
		)
		if err != nil {
			return nil, err
		}
		ch := dal.NewRecord()
		ch.Set("cart", cart)
		ch.Set("additional_data", Get(input, "additional_data"))
		if err := composer.CreateHook(e, "cartCreated", ch); err != nil {
			return nil, err
		}
		return cart, nil
	})

func itemsOrEmpty(items []*dal.Record) []*dal.Record {
	if items == nil {
		return []*dal.Record{}
	}
	return items
}

func asStrings(v any) []string {
	var out []string
	switch t := v.(type) {
	case []any:
		for _, x := range t {
			if s, ok := x.(string); ok {
				out = append(out, s)
			}
		}
	case []string:
		out = t
	}
	return out
}
