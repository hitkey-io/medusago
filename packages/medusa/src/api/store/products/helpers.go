package products

import (
	"context"
	"strconv"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
)

func get(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}

func records(v any) []*dal.Record {
	switch t := v.(type) {
	case []any:
		var out []*dal.Record
		for _, x := range t {
			if r, ok := x.(*dal.Record); ok && r != nil {
				out = append(out, r)
			}
		}
		return out
	case []*dal.Record:
		return t
	}
	return nil
}

// FilterOutInternalProductCategories mirrors filterOutInternalProductCategories.
func FilterOutInternalProductCategories(products []*dal.Record) {
	for _, p := range products {
		v, ok := p.Get("categories")
		if !ok || v == nil {
			continue
		}
		list, ok := v.([]any)
		if !ok {
			continue
		}
		out := []any{}
		for _, c := range list {
			if r, ok := c.(*dal.Record); ok {
				if b, _ := get(r, "is_internal").(bool); b {
					continue
				}
			}
			out = append(out, c)
		}
		p.Set("categories", out)
	}
}

func toNumber(d *totals.BN) any {
	f, err := strconv.ParseFloat(d.String(), 64)
	if err != nil {
		return nil
	}
	return f
}

// CalculateAmountsWithTax mirrors calculateAmountsWithTax.
func CalculateAmountsWithTax(taxLines []*dal.Record, amount any, includesTax bool) (withTax, withoutTax any) {
	rates := make([]any, 0, len(taxLines))
	for _, l := range taxLines {
		rates = append(rates, get(l, "rate"))
	}
	sumTaxRate := totals.Div(totals.Add(rates...), 100)
	var taxable any = amount
	if includesTax {
		taxable = totals.Div(amount, totals.Add(1, sumTaxRate))
	}
	tax := totals.Convert(0)
	if !includesTax {
		for _, l := range taxLines {
			rate := totals.Div(get(l, "rate"), 100)
			tax = totals.Add(tax, totals.Mult(taxable, rate))
		}
	}
	if includesTax {
		return amount, toNumber(totals.Sub(amount, tax))
	}
	return toNumber(totals.Add(tax, amount)), amount
}

type taxLinesService interface {
	GetTaxLines(ctx context.Context, items []*dal.Record, calculationContext *dal.Record) ([]*dal.Record, error)
}

// TaxedVariant is a variant with the product data of its tax item.
type TaxedVariant struct {
	Variant       *dal.Record
	ProductID     any
	ProductTypeID any
}

// TaxContexts returns the tax line context when taxes must be computed.
func TaxContexts(req *medusahttp.MedusaRequest) *dal.Record {
	inclusivity, _ := get(req.TaxContext, "taxInclusivityContext").(*dal.Record)
	lineContext, _ := get(req.TaxContext, "taxLineContext").(*dal.Record)
	if inclusivity == nil || lineContext == nil {
		return nil
	}
	return lineContext
}

// ApplyVariantTaxes runs getTaxLines over the variants with a calculated
// price and sets the *_with_tax / *_without_tax amounts.
func ApplyVariantTaxes(req *medusahttp.MedusaRequest, lineContext *dal.Record, variants []TaxedVariant) error {
	svc, ok := req.Scope.Resolve(utilsmodulessdk.TAX).(taxLinesService)
	if !ok {
		return nil
	}
	var items []*dal.Record
	for _, tv := range variants {
		cp, _ := get(tv.Variant, "calculated_price").(*dal.Record)
		if cp == nil {
			continue
		}
		item := dal.NewRecord()
		item.Set("id", get(tv.Variant, "id"))
		item.Set("product_id", tv.ProductID)
		item.Set("product_type_id", tv.ProductTypeID)
		item.Set("quantity", float64(1))
		item.Set("unit_price", get(cp, "calculated_amount"))
		item.Set("currency_code", get(cp, "currency_code"))
		items = append(items, item)
	}
	lines, err := svc.GetTaxLines(req.Ctx(), items, lineContext)
	if err != nil {
		return err
	}
	byItem := map[any][]*dal.Record{}
	for _, l := range lines {
		k := get(l, "line_item_id")
		byItem[k] = append(byItem[k], l)
	}
	for _, tv := range variants {
		cp, _ := get(tv.Variant, "calculated_price").(*dal.Record)
		if cp == nil {
			continue
		}
		rates := byItem[get(tv.Variant, "id")]
		inc, _ := get(cp, "is_calculated_price_tax_inclusive").(bool)
		with, without := CalculateAmountsWithTax(rates, get(cp, "calculated_amount"), inc)
		cp.Set("calculated_amount_with_tax", with)
		cp.Set("calculated_amount_without_tax", without)
		oinc, _ := get(cp, "is_original_price_tax_inclusive").(bool)
		owith, owithout := CalculateAmountsWithTax(rates, get(cp, "original_amount"), oinc)
		cp.Set("original_amount_with_tax", owith)
		cp.Set("original_amount_without_tax", owithout)
	}
	return nil
}

// WrapProductsWithTaxPrices mirrors wrapProductsWithTaxPrices.
func WrapProductsWithTaxPrices(req *medusahttp.MedusaRequest, products []*dal.Record) error {
	lineContext := TaxContexts(req)
	if lineContext == nil {
		return nil
	}
	inclusivity, _ := get(req.TaxContext, "taxInclusivityContext").(*dal.Record)
	if b, _ := get(inclusivity, "automaticTaxes").(bool); !b {
		return nil
	}
	var variants []TaxedVariant
	for _, p := range products {
		for _, v := range records(get(p, "variants")) {
			variants = append(variants, TaxedVariant{Variant: v, ProductID: get(p, "id"), ProductTypeID: get(p, "type_id")})
		}
	}
	return ApplyVariantTaxes(req, lineContext, variants)
}
