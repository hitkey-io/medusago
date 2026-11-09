// Package steps mirrors @medusajs/core-flows/src/price-list/steps.
package steps

import (
	"context"
	"fmt"
	"strings"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

type pricingService interface {
	CreatePriceLists(ctx context.Context, data []*dal.Record) ([]*dal.Record, error)
	UpdatePriceLists(ctx context.Context, data []*dal.Record) ([]*dal.Record, error)
	AddPriceListPrices(ctx context.Context, data []*dal.Record) ([]*dal.Record, error)
	UpdatePriceListPrices(ctx context.Context, data []*dal.Record) ([]*dal.Record, error)
	RemovePrices(ctx context.Context, ids []any) error
	ListBySuffix(ctx context.Context, suffix string, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]*dal.Record, error)
	Internal(className string) *utilsmodulessdk.InternalService
}

func pricing(c composer.Container) pricingService {
	return c.Resolve(utilsmodulessdk.PRICING).(pricingService)
}

func get(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}

func records(v any) []*dal.Record {
	switch t := v.(type) {
	case []*dal.Record:
		return t
	case []any:
		out := make([]*dal.Record, 0, len(t))
		for _, x := range t {
			if r, ok := x.(*dal.Record); ok {
				out = append(out, r)
			}
		}
		return out
	}
	return nil
}

// VariantPriceMap is variant_id -> price_set_id.
type VariantPriceMap map[string]any

// ValidateVariantPriceLinksStep mirrors validateVariantPriceLinksStep.
var ValidateVariantPriceLinksStep = composer.CreateStep("validate-variant-price-links",
	func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[VariantPriceMap], error) {
		if len(data) == 0 {
			return composer.NewStepResponse(VariantPriceMap(nil)), nil
		}
		var variantIDs []any
		for _, pl := range data {
			for _, p := range records(get(pl, "prices")) {
				if v := get(p, "variant_id"); v != nil {
					variantIDs = append(variantIDs, v)
				}
			}
		}
		if variantIDs == nil {
			variantIDs = []any{}
		}
		vars := dal.NewRecord()
		vars.Set("variant_id", variantIDs)
		links, err := commonsteps.RemoteQuery(ctx, "product_variant_price_set", vars, []string{"variant_id", "price_set_id"})
		if err != nil {
			return composer.StepResponse[VariantPriceMap]{}, err
		}
		m := VariantPriceMap{}
		for _, l := range links {
			r, _ := l.(*dal.Record)
			m[fmt.Sprint(get(r, "variant_id"))] = get(r, "price_set_id")
		}
		var without []string
		for _, id := range variantIDs {
			if v := m[fmt.Sprint(id)]; v == nil || v == "" {
				without = append(without, fmt.Sprint(id))
			}
		}
		if len(without) > 0 {
			return composer.StepResponse[VariantPriceMap]{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "No price set exist for variants: "+strings.Join(without, ", "))
		}
		return composer.NewStepResponse(m), nil
	}, nil)

// ValidatePriceListsStep mirrors validatePriceListsStep.
var ValidatePriceListsStep = composer.CreateStep("validate-price-lists",
	func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[any], error) {
		if len(data) == 0 {
			return composer.NewStepResponse[any](nil), nil
		}
		ids := make([]any, len(data))
		for i, d := range data {
			ids[i] = get(d, "id")
		}
		f := dal.NewRecord()
		f.Set("id", ids)
		lists, err := pricing(ctx.Container).ListBySuffix(ctx.Context, "PriceLists", f, utilsmodulessdk.FindConfig{})
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		have := map[string]bool{}
		for _, l := range lists {
			have[fmt.Sprint(get(l, "id"))] = true
		}
		var diff []string
		for _, id := range ids {
			if !have[fmt.Sprint(id)] {
				diff = append(diff, fmt.Sprint(id))
			}
		}
		if len(diff) > 0 {
			return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("Price lists with id: %s was not found", strings.Join(diff, ", ")))
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// PriceListsStepInput is { data, variant_price_map }.
type PriceListsStepInput struct {
	Data            []*dal.Record
	VariantPriceMap VariantPriceMap
}

// CreatePriceListsStep mirrors createPriceListsStep.
var CreatePriceListsStep = composer.CreateStep("create-price-lists",
	func(ctx composer.StepExecutionContext, in PriceListsStepInput) (composer.StepResponse[[]*dal.Record], error) {
		if len(in.Data) == 0 {
			return composer.NewStepResponse([]*dal.Record{}), nil
		}
		create := make([]*dal.Record, len(in.Data))
		for i, pl := range in.Data {
			c := dal.NewRecord()
			for _, k := range pl.Keys() {
				if k == "prices" {
					continue
				}
				v, _ := pl.Get(k)
				c.Set(k, v)
			}
			var prices []any
			for _, p := range records(get(pl, "prices")) {
				n := dal.NewRecord()
				n.Set("currency_code", get(p, "currency_code"))
				n.Set("amount", get(p, "amount"))
				n.Set("min_quantity", get(p, "min_quantity"))
				n.Set("max_quantity", get(p, "max_quantity"))
				n.Set("price_set_id", in.VariantPriceMap[fmt.Sprint(get(p, "variant_id"))])
				if r, ok := p.Get("rules"); ok {
					n.Set("rules", r)
				}
				prices = append(prices, n)
			}
			if prices == nil {
				prices = []any{}
			}
			c.Set("prices", prices)
			create[i] = c
		}
		created, err := pricing(ctx.Container).CreatePriceLists(ctx.Context, create)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(created, any(commonsteps.RecordIDs(created))), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		ids, _ := data.([]any)
		if len(ids) == 0 {
			return nil
		}
		_, err := pricing(ctx.Container).Internal("PriceList").Delete(ctx.Context, ids)
		return err
	})

type updateCompensation struct {
	before []*dal.Record
	keys   map[string]bool
}

// UpdatePriceListsStep mirrors updatePriceListsStep.
var UpdatePriceListsStep = composer.CreateStep("update-price-lists",
	func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		if len(data) == 0 {
			return composer.NewStepResponse[[]*dal.Record](nil), nil
		}
		svc := pricing(ctx.Container)
		ids := make([]any, len(data))
		keys := map[string]bool{}
		for i, d := range data {
			ids[i] = get(d, "id")
			for _, k := range d.Keys() {
				keys[k] = true
			}
		}
		f := dal.NewRecord()
		f.Set("id", ids)
		before, err := svc.ListBySuffix(ctx.Context, "PriceLists", f, utilsmodulessdk.FindConfig{Relations: []string{"price_list_rules"}})
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		updated, err := svc.UpdatePriceLists(ctx.Context, data)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(updated, any(updateCompensation{before: before, keys: keys})), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		c, ok := data.(updateCompensation)
		if !ok || len(c.before) == 0 {
			return nil
		}
		restore := make([]*dal.Record, len(c.before))
		for i, b := range c.before {
			r := dal.NewRecord()
			r.Set("id", get(b, "id"))
			for _, k := range b.Keys() {
				if c.keys[k] && k != "id" {
					v, _ := b.Get(k)
					r.Set(k, v)
				}
			}
			if c.keys["rules"] {
				rules := dal.NewRecord()
				for _, rule := range records(get(b, "price_list_rules")) {
					rules.Set(fmt.Sprint(get(rule, "attribute")), get(rule, "value"))
				}
				r.Set("rules", rules)
			}
			restore[i] = r
		}
		_, err := pricing(ctx.Container).UpdatePriceLists(ctx.Context, restore)
		return err
	})

// DeletePriceListsStep mirrors deletePriceListsStep.
var DeletePriceListsStep = commonsteps.SoftDeleteRecordsStep("delete-price-lists", utilsmodulessdk.PRICING, "PriceList")

// priceListPrices builds the module input { price_list_id, prices } with
// variant ids replaced by price set ids.
func priceListPrices(in PriceListsStepInput, skipEmpty bool) ([]*dal.Record, []any) {
	var out []*dal.Record
	var priceIDs []any
	for _, d := range in.Data {
		var prices []any
		for _, p := range records(get(d, "prices")) {
			n := dal.NewRecord()
			for _, k := range p.Keys() {
				if k == "variant_id" {
					continue
				}
				v, _ := p.Get(k)
				n.Set(k, v)
			}
			n.Set("price_set_id", in.VariantPriceMap[fmt.Sprint(get(p, "variant_id"))])
			prices = append(prices, n)
			if id := get(p, "id"); id != nil && id != "" {
				priceIDs = append(priceIDs, id)
			}
		}
		if skipEmpty && len(prices) == 0 {
			continue
		}
		if prices == nil {
			prices = []any{}
		}
		r := dal.NewRecord()
		r.Set("price_list_id", get(d, "id"))
		r.Set("prices", prices)
		out = append(out, r)
	}
	return out, priceIDs
}

// CreatePriceListPricesStep mirrors createPriceListPricesStep.
var CreatePriceListPricesStep = composer.CreateStep("create-price-list-prices",
	func(ctx composer.StepExecutionContext, in PriceListsStepInput) (composer.StepResponse[[]*dal.Record], error) {
		data, _ := priceListPrices(in, true)
		if len(data) == 0 {
			return composer.NewStepResponse([]*dal.Record{}), nil
		}
		created, err := pricing(ctx.Container).AddPriceListPrices(ctx.Context, data)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(created, any(commonsteps.RecordIDs(created))), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		ids, _ := data.([]any)
		if len(ids) == 0 {
			return nil
		}
		return pricing(ctx.Container).RemovePrices(ctx.Context, ids)
	})

// UpdatePriceListPricesStep mirrors updatePriceListPricesStep (the
// upstream compensation data is always empty: its map is iterated with
// Object.entries).
var UpdatePriceListPricesStep = composer.CreateStep("update-price-list-prices",
	func(ctx composer.StepExecutionContext, in PriceListsStepInput) (composer.StepResponse[[]*dal.Record], error) {
		data, _ := priceListPrices(in, false)
		if len(data) == 0 {
			return composer.NewStepResponse([]*dal.Record{}), nil
		}
		updated, err := pricing(ctx.Container).UpdatePriceListPrices(ctx.Context, data)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(updated), nil
	}, nil)

// RemovePriceListPricesStep mirrors removePriceListPricesStep.
var RemovePriceListPricesStep = composer.CreateStep("remove-price-list-prices",
	func(ctx composer.StepExecutionContext, ids []any) (composer.StepResponse[[]any], error) {
		if len(ids) == 0 {
			return composer.NewStepResponse([]any{}, any([]any{})), nil
		}
		svc := pricing(ctx.Container)
		f := dal.NewRecord()
		f.Set("id", ids)
		prices, err := svc.Internal("Price").List(ctx.Context, f, utilsmodulessdk.FindConfig{})
		if err != nil {
			return composer.StepResponse[[]any]{}, err
		}
		priceIDs := commonsteps.RecordIDs(prices)
		if len(priceIDs) > 0 {
			if _, err := svc.Internal("Price").SoftDelete(ctx.Context, priceIDs); err != nil {
				return composer.StepResponse[[]any]{}, err
			}
		}
		return composer.NewStepResponse(priceIDs, any(priceIDs)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		ids, _ := data.([]any)
		if len(ids) == 0 {
			return nil
		}
		_, err := pricing(ctx.Container).Internal("Price").Restore(ctx.Context, ids)
		return err
	})
