// Package cart mirrors totals/cart.
package cart

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
	creditlines "github.com/hitkey-io/medusago/packages/core/utils/src/totals/credit-lines"
	lineitem "github.com/hitkey-io/medusago/packages/core/utils/src/totals/line-item"
	shippingmethod "github.com/hitkey-io/medusago/packages/core/utils/src/totals/shipping-method"
)

// OptionalFields mirrors the optionalFields of decorateCartTotals.
var OptionalFields = [][2]string{
	{"detail.fulfilled_quantity", "fulfilled_total"},
	{"detail.shipped_quantity", "shipped_total"},
	{"detail.return_requested_quantity", "return_requested_total"},
	{"detail.return_received_quantity", "return_received_total"},
	{"detail.return_dismissed_quantity", "return_dismissed_total"},
	{"detail.written_off_quantity", "write_off_total"},
}

func field(o totals.Object, k string) any {
	v, _ := o.Get(k)
	return v
}

func totalsFor(m map[any]*lineitem.Totals, o totals.Object, index int) *lineitem.Totals {
	if id, ok := o.Get("id"); ok && id != nil {
		if t := m[id]; t != nil {
			return t
		}
		return lineitem.NewTotals()
	}
	if t := m[index]; t != nil {
		return t
	}
	return lineitem.NewTotals()
}

// DecorateCartTotals mirrors decorateCartTotals.
func DecorateCartTotals(cart totals.Object, includeTaxes bool) totals.Object {
	totals.TransformPropertiesToBigNumber(cart, nil, nil)

	creditLines := totals.Objects(field(cart, "credit_lines"))
	items := totals.Objects(field(cart, "items"))
	methods := totals.Objects(field(cart, "shipping_methods"))
	region := totals.AsObject(field(cart, "region"))
	includeTax := includeTaxes || (region != nil && totals.Truthy(field(region, "automatic_taxes")))

	itemsTotals := lineitem.GetLineItemsTotals(items, lineitem.Context{IncludeTax: includeTax, ExtraQuantityFields: OptionalFields})
	methodsTotals := shippingmethod.GetShippingMethodsTotals(methods, includeTax)

	extraKeys := []string{}
	extra := map[string]*totals.BN{}
	creditLinesSumTaxRate := totals.Div(totals.Convert(0), 100)

	zero := func() *totals.BN { return totals.Convert(0) }
	subtotal, discountTotal, discountSubtotal, discountTaxTotal := zero(), zero(), zero(), zero()
	itemsSubtotal, itemsTotal, itemsOriginalTotal, itemsOriginalSubtotal, itemsTaxTotal, itemsOriginalTaxTotal, itemsDiscountTotal := zero(), zero(), zero(), zero(), zero(), zero(), zero()
	shippingSubtotal, shippingTotal, shippingOriginalTotal, shippingOriginalSubtotal, shippingTaxTotal, shippingOriginalTaxTotal, shippingDiscountTotal := zero(), zero(), zero(), zero(), zero(), zero(), zero()

	for i, item := range items {
		t := totalsFor(itemsTotals, item, i)
		t.Assign(item)
		itemSubtotal := t.Get("subtotal")
		subtotal = totals.Add(subtotal, itemSubtotal)
		discountTotal = totals.Add(discountTotal, t.Get("discount_total"))
		discountSubtotal = totals.Add(discountSubtotal, t.Get("discount_subtotal"))
		discountTaxTotal = totals.Add(discountTaxTotal, t.Get("discount_tax_total"))
		itemsTotal = totals.Add(itemsTotal, t.Get("total"))
		itemsOriginalTotal = totals.Add(itemsOriginalTotal, t.Get("original_total"))
		itemsOriginalSubtotal = totals.Add(itemsOriginalSubtotal, itemSubtotal)
		itemsSubtotal = totals.Add(itemsSubtotal, itemSubtotal)
		itemsTaxTotal = totals.Add(itemsTaxTotal, t.Get("tax_total"))
		itemsOriginalTaxTotal = totals.Add(itemsOriginalTaxTotal, t.Get("original_tax_total"))
		itemsDiscountTotal = totals.Add(itemsDiscountTotal, t.Get("discount_total"))
		for _, f := range OptionalFields {
			key := f[1]
			if v, ok := item.Get(key); ok {
				if _, seen := extra[key]; !seen {
					extra[key] = zero()
					extraKeys = append(extraKeys, key)
				}
				if v == nil {
					v = 0
				}
				extra[key] = totals.Add(extra[key], v)
			}
		}
	}

	for i, m := range methods {
		t := totalsFor(methodsTotals, m, i)
		t.Assign(m)
		subtotal = totals.Add(subtotal, t.Get("subtotal"))
		discountTotal = totals.Add(discountTotal, t.Get("discount_total"))
		discountSubtotal = totals.Add(discountSubtotal, t.Get("discount_subtotal"))
		discountTaxTotal = totals.Add(discountTaxTotal, t.Get("discount_tax_total"))
		shippingSubtotal = totals.Add(shippingSubtotal, t.Get("subtotal"))
		shippingTotal = totals.Add(shippingTotal, t.Get("total"))
		shippingOriginalTotal = totals.Add(shippingOriginalTotal, t.Get("original_total"))
		shippingOriginalSubtotal = totals.Add(shippingOriginalSubtotal, t.Get("subtotal"))
		shippingTaxTotal = totals.Add(shippingTaxTotal, t.Get("tax_total"))
		shippingOriginalTaxTotal = totals.Add(shippingOriginalTaxTotal, t.Get("original_tax_total"))
		shippingDiscountTotal = totals.Add(shippingDiscountTotal, t.Get("discount_total"))
	}

	currency, _ := field(cart, "currency_code").(string)
	clTotal, clSubtotal, clTaxTotal := creditlines.CalculateCreditLinesTotal(creditLines, false, creditLinesSumTaxRate, currency)

	taxTotal := totals.Add(itemsTaxTotal, shippingTaxTotal)
	originalTaxTotal := totals.Add(itemsOriginalTaxTotal, shippingOriginalTaxTotal)
	originalTotal := totals.Add(itemsOriginalTotal, shippingOriginalTotal)
	originalSubtotal := totals.Add(itemsOriginalSubtotal, shippingOriginalSubtotal)
	total := totals.Sub(totals.Add(subtotal, taxTotal), discountSubtotal, clTotal)

	set := func(k string, v any) { cart.Set(k, totals.NewBN(v)) }
	set("total", total)
	set("subtotal", subtotal)
	set("tax_total", taxTotal)
	set("discount_total", discountTotal)
	set("discount_subtotal", discountSubtotal)
	set("discount_tax_total", discountTaxTotal)
	set("credit_line_total", clTotal)
	set("credit_line_subtotal", clSubtotal)
	set("credit_line_tax_total", clTaxTotal)
	set("original_total", originalTotal)
	set("original_subtotal", originalSubtotal)
	set("original_tax_total", originalTaxTotal)

	if v, ok := cart.Get("items"); ok && v != nil {
		set("item_total", itemsTotal)
		set("item_subtotal", itemsSubtotal)
		set("item_tax_total", itemsTaxTotal)
		set("item_discount_total", itemsDiscountTotal)
		set("original_item_total", itemsOriginalTotal)
		set("original_item_subtotal", itemsOriginalSubtotal)
		set("original_item_tax_total", itemsOriginalTaxTotal)
		for _, k := range extraKeys {
			set(k, extra[k])
		}
	}
	if v, ok := cart.Get("shipping_methods"); ok && v != nil {
		set("shipping_total", shippingTotal)
		set("shipping_subtotal", shippingSubtotal)
		set("shipping_tax_total", shippingTaxTotal)
		set("shipping_discount_total", shippingDiscountTotal)
		set("original_shipping_tax_total", shippingOriginalTaxTotal)
		set("original_shipping_subtotal", shippingOriginalSubtotal)
		set("original_shipping_total", shippingOriginalTotal)
	}

	if summary := totals.AsObject(field(cart, "summary")); summary != nil {
		pending := []any{0}
		for _, item := range items {
			v, _ := item.Get("return_requested_total")
			if v == nil {
				v = 0
			}
			pending = append(pending, v)
		}
		if len(items) == 0 {
			pending = append(pending, 0)
		}
		pendingReturnTotal := totals.Add(pending...)
		tt, _ := summary.Get("transaction_total")
		if tt == nil {
			tt = 0
		}
		diff := totals.NewBN(totals.Sub(totals.Sub(field(cart, "total"), pendingReturnTotal), tt))
		if totals.Lte(totals.Abs(diff), creditlines.CurrencyEpsilon(currency)) {
			summary.Set("pending_difference", totals.Convert(0))
		} else {
			summary.Set("pending_difference", diff)
		}
	}
	return cart
}
