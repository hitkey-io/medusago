package workflows

import (
	cartsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/steps"
	cartutils "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/utils"
	cartworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/workflows"
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	draftordersteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/draft-order/steps"
	ordersteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/order/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

var createOrderFreshFields = []string{"shipping_address.*", "billing_address.*", "summary.*", "items.*", "credit_lines.*", "items.tax_lines.*",
	"items.adjustments.*", "shipping_methods.*", "shipping_methods.tax_lines.*", "shipping_methods.adjustments.*", "transactions.*", "currency_code",
	"items.tax_lines.*", "items.adjustments.*", "shipping_methods.tax_lines.*", "shipping_methods.adjustments.*", "total", "id", "email", "customer_id"}

func copyRec(r *dal.Record) *dal.Record {
	out := dal.NewRecord()
	if r == nil {
		return out
	}
	for _, k := range r.Keys() {
		v, _ := r.Get(k)
		out.Set(k, v)
	}
	return out
}

// prepareOrderLineItems mirrors prepareLineItems of createOrderWorkflow.
func prepareOrderLineItems(items, variants []*dal.Record) ([]*dal.Record, error) {
	out := make([]*dal.Record, 0, len(items))
	for _, item := range items {
		var variant *dal.Record
		for _, v := range variants {
			if g(v, "id") == g(item, "variant_id") {
				variant = v
				break
			}
		}
		cp := rec(g(variant, "calculated_price"))
		inclusive := g(item, "is_tax_inclusive")
		if inclusive == nil {
			inclusive = g(cp, "is_calculated_price_tax_inclusive")
		}
		in := cartutils.PrepareLineItemDataInput{
			Item:           item,
			Variant:        variant,
			UnitPrice:      g(item, "unit_price"),
			IsTaxInclusive: truthy(inclusive),
			IsCustomPrice:  g(item, "unit_price") != nil,
			TaxLines:       recs(g(item, "tax_lines")),
			Adjustments:    recs(g(item, "adjustments")),
		}
		if in.TaxLines == nil {
			in.TaxLines = []*dal.Record{}
		}
		if in.Adjustments == nil {
			in.Adjustments = []*dal.Record{}
		}
		if variant != nil && in.UnitPrice == nil {
			in.UnitPrice = g(cp, "calculated_amount")
		}
		li, err := cartutils.PrepareLineItemData(in)
		if err != nil {
			return nil, err
		}
		out = append(out, li)
	}
	return out, nil
}

// orderInputFor mirrors getOrderInput.
func orderInputFor(input, region *dal.Record, customer *dal.Record, salesChannel *dal.Record) *dal.Record {
	data := copyRec(input)
	for _, k := range []string{"shipping_address", "billing_address"} {
		addr := copyRec(rec(g(input, k)))
		addr.Delete("id")
		if addr.Len() > 0 {
			data.Set(k, addr)
		} else if data.Has(k) {
			data.Delete(k)
		}
	}
	currency := g(input, "currency_code")
	if currency == nil {
		currency = g(region, "currency_code")
	}
	data.Set("currency_code", currency)
	data.Set("region_id", g(region, "id"))
	if id := g(customer, "id"); truthy(id) {
		data.Set("customer_id", id)
		email := g(input, "email")
		if email == nil {
			email = g(customer, "email")
		}
		data.Set("email", email)
	}
	if id := g(salesChannel, "id"); truthy(id) {
		data.Set("sales_channel_id", id)
	}
	return data
}

// CreateOrderWorkflow mirrors createOrderWorkflow.
var CreateOrderWorkflow = composer.CreateWorkflow("create-orders",
	func(e *composer.Execution, input *dal.Record) (*dal.Record, error) {
		items := recs(g(input, "items"))
		var variantIDs []any
		for _, item := range items {
			if v := g(item, "variant_id"); truthy(v) {
				variantIDs = append(variantIDs, v)
			}
		}
		var salesChannel, region *dal.Record
		var customerData cartsteps.CustomerData
		regionID, _ := g(input, "region_id").(string)
		err := composer.Parallelize(
			func() error {
				var err error
				salesChannel, err = cartsteps.FindSalesChannelStep.Run(e, g(input, "sales_channel_id"))
				return err
			},
			func() error {
				var err error
				region, err = cartsteps.FindOneOrAnyRegionStep.Run(e, regionID)
				return err
			},
			func() error {
				var err error
				customerData, err = cartsteps.FindOrCreateCustomerStep.Run(e, cartsteps.FindOrCreateCustomerInput{CustomerID: g(input, "customer_id"), Email: g(input, "email")})
				return err
			},
		)
		if err != nil {
			return nil, err
		}
		hookIn := dal.NewRecord()
		hookIn.Set("variantIds", variantIDs)
		hookIn.Set("region", region)
		hookIn.Set("customerData", customerData)
		hookIn.Set("additional_data", g(input, "additional_data"))
		if err := composer.CreateHook(e, "setPricingContext", hookIn); err != nil {
			return nil, err
		}
		customerID := g(customerData.Customer, "id")
		var customerForPricing *dal.Record
		if truthy(customerID) {
			f := dal.NewRecord()
			f.Set("id", customerID)
			isList := false
			res, err := commonsteps.UseQueryGraphStep.Config("customer-groups-query", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
				GraphInput: query.GraphInput{Entity: "customer", Fields: []string{"id", "groups.id"}, Filters: f},
				IsList:     &isList,
			})
			if err != nil {
				return nil, err
			}
			customerForPricing = rec(res.Data)
		}
		var forCalculation, withoutCalculation []any
		for _, item := range items {
			if truthy(g(item, "variant_id")) && (g(item, "unit_price") == nil || g(item, "is_tax_inclusive") == nil) {
				forCalculation = append(forCalculation, g(item, "variant_id"))
			} else if v := g(item, "variant_id"); v != nil {
				withoutCalculation = append(withoutCalculation, v)
			}
		}
		variantFields := cartutils.Deduplicate(cartutils.ProductVariantsFields, cartutils.RequiredVariantFieldsForInventoryConfirmation)
		wf := dal.NewRecord()
		if withoutCalculation == nil {
			withoutCalculation = []any{}
		}
		wf.Set("id", withoutCalculation)
		without, err := queryList(e, "query-variants-without-calculated-price", "variants", wf, variantFields)
		if err != nil {
			return nil, err
		}
		var calculated *cartworkflows.VariantsAndItems
		if len(forCalculation) > 0 {
			cart := dal.NewRecord()
			cart.Set("currency_code", g(input, "currency_code"))
			cart.Set("region", region)
			cart.Set("region_id", g(region, "id"))
			if truthy(customerID) {
				cart.Set("customer_id", customerID)
			}
			if customerForPricing != nil {
				cart.Set("customer", customerForPricing)
			}
			vi, err := cartworkflows.GetVariantsAndItemsWithPrices.RunAsStep(e, cartworkflows.VariantsAndItemsInput{
				Cart: cart, Items: items, VariantIDs: forCalculation, VariantFields: variantFields,
			})
			if err != nil {
				return nil, err
			}
			calculated = &vi
		}
		variants := append([]*dal.Record{}, without...)
		if calculated != nil {
			variants = append(variants, calculated.Variants...)
		}
		channelID, _ := g(salesChannel, "id").(string)
		if _, err := cartworkflows.ConfirmVariantInventoryWorkflow.RunAsStep(e, cartutils.ConfirmVariantInventoryInput{SalesChannelID: channelID, Variants: variants, Items: items}); err != nil {
			return nil, err
		}
		orderInput := orderInputFor(input, region, customerData.Customer, salesChannel)

		var withCalc []*dal.Record
		if calculated != nil {
			for _, li := range calculated.LineItems {
				withCalc = append(withCalc, rec(g(li, "data")))
			}
		}
		var rest []*dal.Record
		for _, item := range items {
			found := false
			for _, d := range withCalc {
				if (g(d, "id") != nil && g(d, "id") == g(item, "id")) || (g(item, "variant_id") != nil && g(d, "variant_id") == g(item, "variant_id")) ||
					(g(d, "title") == g(item, "title") && g(d, "subtitle") == g(item, "subtitle") && g(d, "thumbnail") == g(item, "thumbnail")) {
					found = true
					break
				}
			}
			if !found {
				rest = append(rest, item)
			}
		}
		lineItems := withCalc
		if len(rest) > 0 {
			prepared, err := prepareOrderLineItems(rest, without)
			if err != nil {
				return nil, err
			}
			lineItems = append(prepared, withCalc...)
		}
		if lineItems == nil {
			lineItems = []*dal.Record{}
		}
		if _, err := cartsteps.ValidateLineItemPricesStep.Run(e, lineItems); err != nil {
			return nil, err
		}
		toCreate := copyRec(orderInput)
		toCreate.Set("items", lineItems)
		orders, err := ordersteps.CreateOrdersStep.Run(e, []*dal.Record{toCreate})
		if err != nil {
			return nil, err
		}
		order := orders[0]
		var promoCodes []string
		for _, c := range anyList(g(input, "promo_codes")) {
			if s, ok := c.(string); ok {
				promoCodes = append(promoCodes, s)
			}
		}
		if promoCodes == nil {
			promoCodes = []string{}
		}
		ff := dal.NewRecord()
		ff.Set("id", g(order, "id"))
		isList := false
		fres, err := commonsteps.UseQueryGraphStep.Config("query-fresh-order", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
			GraphInput: query.GraphInput{Entity: "orders", Fields: createOrderFreshFields, Filters: ff},
			IsList:     &isList,
		})
		if err != nil {
			return nil, err
		}
		fresh := rec(fres.Data)
		err = composer.Parallelize(
			func() error {
				_, err := UpdateOrderTaxLinesWorkflow.RunAsStep(e, UpdateOrderTaxLinesInput{OrderID: g(order, "id")})
				return err
			},
			func() error {
				_, err := draftordersteps.RefreshDraftOrderAdjustmentsWorkflow.RunAsStep(e, draftordersteps.RefreshDraftOrderAdjustmentsInput{Order: fresh, PromoCodes: promoCodes, Action: cartsteps.PromotionActionReplace})
				return err
			},
		)
		if err != nil {
			return nil, err
		}
		created := dal.NewRecord()
		created.Set("order", fresh)
		created.Set("additional_data", g(input, "additional_data"))
		if err := composer.CreateHook(e, "orderCreated", created); err != nil {
			return nil, err
		}
		return order, nil
	})

func anyList(v any) []any {
	switch t := v.(type) {
	case []any:
		return t
	case []string:
		out := make([]any, len(t))
		for i, s := range t {
			out[i] = s
		}
		return out
	}
	return nil
}
