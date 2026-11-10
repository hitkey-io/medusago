package workflows

import (
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	ordersteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/order/steps"
	taxsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/tax/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

var taxShippingAddressFields = []string{"shipping_address.id", "shipping_address.address_1", "shipping_address.address_2", "shipping_address.city",
	"shipping_address.postal_code", "shipping_address.country_code", "shipping_address.region_code", "shipping_address.province"}

var completeOrderTaxFields = append([]string{"id", "currency_code", "email", "locale", "region.id", "region.automatic_taxes",
	"items.id", "items.is_tax_inclusive", "items.is_giftcard", "items.variant_id", "items.product_id", "items.product_title",
	"items.product_description", "items.product_subtitle", "items.product_type", "items.product_type_id", "items.product_collection",
	"items.product_handle", "items.variant_sku", "items.variant_barcode", "items.variant_title", "items.title", "items.quantity",
	"items.unit_price", "items.tax_lines.id", "items.tax_lines.description", "items.tax_lines.code", "items.tax_lines.rate",
	"items.tax_lines.provider_id", "shipping_methods.id", "shipping_methods.name", "shipping_methods.is_tax_inclusive",
	"shipping_methods.shipping_option_id", "shipping_methods.amount", "shipping_methods.tax_lines.id",
	"shipping_methods.tax_lines.description", "shipping_methods.tax_lines.code", "shipping_methods.tax_lines.rate",
	"shipping_methods.tax_lines.provider_id", "customer.id", "customer.email", "customer.groups.id"}, taxShippingAddressFields...)

var orderTaxFields = append([]string{"id", "currency_code", "email", "locale", "region.id", "region.automatic_taxes",
	"shipping_methods.id", "shipping_methods.name", "shipping_methods.tax_lines.id", "shipping_methods.tax_lines.description",
	"shipping_methods.tax_lines.code", "shipping_methods.tax_lines.rate", "shipping_methods.tax_lines.provider_id",
	"shipping_methods.shipping_option_id", "shipping_methods.amount", "customer.id", "customer.email", "customer.groups.id"}, taxShippingAddressFields...)

var shippingMethodTaxFields = []string{"id", "name", "shipping_option_id", "is_tax_inclusive", "amount", "tax_lines.id",
	"tax_lines.description", "tax_lines.code", "tax_lines.rate", "tax_lines.provider_id"}

var lineItemTaxFields = []string{"id", "variant_id", "product_id", "is_tax_inclusive", "is_giftcard", "product_title",
	"product_description", "product_subtitle", "product_type", "product_type_id", "product_collection", "product_handle",
	"variant_sku", "variant_barcode", "variant_title", "title", "quantity", "unit_price", "tax_lines.id", "tax_lines.description",
	"tax_lines.code", "tax_lines.rate", "tax_lines.provider_id"}

// UpdateOrderTaxLinesInput mirrors UpdateOrderTaxLinesWorkflowInput.
type UpdateOrderTaxLinesInput struct {
	OrderID             any
	ItemIDs             []any
	ShippingMethodIDs   []any
	ForceTaxCalculation bool
	IsReturn            bool
	ShippingAddress     *dal.Record
}

func queryList(e *composer.Execution, name, entity string, filters *dal.Record, fields []string) ([]*dal.Record, error) {
	res, err := commonsteps.UseQueryGraphStep.Config(name, composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
		GraphInput: query.GraphInput{Entity: entity, Fields: fields, Filters: filters},
	})
	if err != nil {
		return nil, err
	}
	return recs(res.Data), nil
}

// UpdateOrderTaxLinesWorkflow mirrors updateOrderTaxLinesWorkflow.
var UpdateOrderTaxLinesWorkflow = composer.CreateWorkflow("update-order-tax-lines",
	func(e *composer.Execution, in UpdateOrderTaxLinesInput) (*taxsteps.ItemTaxLines, error) {
		isFullOrder := in.ItemIDs == nil && in.ShippingMethodIDs == nil
		fields := orderTaxFields
		if isFullOrder {
			fields = completeOrderTaxFields
		}
		f := dal.NewRecord()
		f.Set("id", in.OrderID)
		isList := false
		res, err := commonsteps.UseQueryGraphStep.Config("order-query", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
			GraphInput: query.GraphInput{Entity: "order", Fields: fields, Filters: f},
			IsList:     &isList,
		})
		if err != nil {
			return nil, err
		}
		order := rec(res.Data)
		var items, methods []*dal.Record
		if len(in.ItemIDs) > 0 {
			f := dal.NewRecord()
			f.Set("id", in.ItemIDs)
			if items, err = queryList(e, "query-order-line-items", "order_line_item", f, lineItemTaxFields); err != nil {
				return nil, err
			}
		}
		if len(in.ShippingMethodIDs) > 0 {
			f := dal.NewRecord()
			f.Set("id", in.ShippingMethodIDs)
			if methods, err = queryList(e, "query-order-shipping-methods", "order_shipping_method", f, shippingMethodTaxFields); err != nil {
				return nil, err
			}
		}
		hookIn := dal.NewRecord()
		hookIn.Set("order", order)
		hookIn.Set("items", items)
		hookIn.Set("shipping_methods", methods)
		if err := composer.CreateHook(e, "setTaxLineContext", hookIn); err != nil {
			return nil, err
		}
		if isFullOrder {
			items = recs(g(order, "items"))
			methods = recs(g(order, "shipping_methods"))
		}
		lines, err := taxsteps.GetItemTaxLinesStep.Run(e, taxsteps.GetItemTaxLinesInput{
			OrderOrCart:         order,
			Items:               items,
			ShippingMethods:     methods,
			ForceTaxCalculation: in.ForceTaxCalculation,
			IsReturn:            in.IsReturn,
			ShippingAddress:     in.ShippingAddress,
		})
		if err != nil {
			return nil, err
		}
		if _, err := ordersteps.SetOrderTaxLinesForItemsStep.Run(e, ordersteps.SetOrderTaxLinesInput{
			Order: order, ItemTaxLines: lines.LineItemTaxLines, ShippingTaxLines: lines.ShippingMethodsTaxLines, IsFullReplacement: isFullOrder,
		}); err != nil {
			return nil, err
		}
		return &lines, nil
	})
