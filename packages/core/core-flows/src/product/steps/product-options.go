package steps

import (
	"context"
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	"github.com/hitkey-io/medusago/packages/modules/product/src/services"
)

// CreateProductOptionsStep mirrors createProductOptionsStep.
var CreateProductOptionsStep = createStep("create-product-options", "ProductOption",
	func(ctx context.Context, s *services.ProductModuleService, data []*dal.Record) ([]*dal.Record, error) {
		return s.CreateProductOptions(ctx, data)
	})

// UpdateProductOptionsStep mirrors updateProductOptionsStep.
var UpdateProductOptionsStep = composer.CreateStep("update-product-options",
	func(ctx composer.StepExecutionContext, in UpdateBySelectorInput) (composer.StepResponse[[]*dal.Record], error) {
		svc := ProductService(ctx.Container)
		cleaned := dal.NewRecord()
		if in.Update != nil {
			for _, k := range in.Update.Keys() {
				if k == "ranks" {
					continue
				}
				v, _ := in.Update.Get(k)
				cleaned.Set(k, v)
			}
		}
		selects, _ := common.GetSelectsAndRelationsFromObjectArray([]common.ObjectLike{cleaned}, common.SelectsAndRelationsOptions{}, "")
		prev, err := svc.ListBySuffix(ctx.Context, "ProductOptions", in.Selector, utilsmodulessdk.FindConfig{Select: selects, Relations: []string{"values"}})
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		if len(prev) == 0 {
			id, _ := in.Selector.Get("id")
			return composer.StepResponse[[]*dal.Record]{}, common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf(`Product option with id "%v" not found`, id))
		}
		out, err := svc.UpdateProductOptions(ctx.Context, in.Selector, in.Update)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(out, any(prev)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		prev, _ := data.([]*dal.Record)
		if len(prev) == 0 {
			return nil
		}
		in := make([]*dal.Record, len(prev))
		for i, o := range prev {
			r := dal.NewRecord()
			for _, k := range o.Keys() {
				v, _ := o.Get(k)
				r.Set(k, v)
			}
			if vs, ok := o.Get("values"); ok {
				var vals []any
				for _, v := range anyList(vs) {
					if vr, ok := v.(*dal.Record); ok {
						x, _ := vr.Get("value")
						vals = append(vals, x)
					}
				}
				r.Set("values", vals)
			}
			in[i] = r
		}
		_, err := ProductService(ctx.Container).UpsertProductOptions(ctx.Context, in)
		return err
	})

func anyList(v any) []any {
	switch t := v.(type) {
	case []any:
		return t
	case []*dal.Record:
		out := make([]any, len(t))
		for i, r := range t {
			out[i] = r
		}
		return out
	}
	return nil
}

// AddProductOptionsToProductStep mirrors addProductOptionsToProductStep.
var AddProductOptionsToProductStep = composer.CreateStep("add-product-options-to-product",
	func(ctx composer.StepExecutionContext, pairs []services.ProductOptionProductPair) (composer.StepResponse[any], error) {
		if len(pairs) == 0 {
			return composer.NewStepResponse[any](nil, any([]services.ProductOptionProductPair{})), nil
		}
		if err := ProductService(ctx.Container).AddProductOptionToProduct(ctx.Context, pairs); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil, any(pairs)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		pairs, _ := data.([]services.ProductOptionProductPair)
		if len(pairs) == 0 {
			return nil
		}
		_, err := ProductService(ctx.Container).RemoveProductOptionFromProduct(ctx.Context, pairs)
		return err
	})

type removedOptions struct {
	links   []services.ProductOptionProductPair
	deleted []any
}

// RemoveProductOptionsFromProductStep mirrors
// removeProductOptionsFromProductStep.
var RemoveProductOptionsFromProductStep = composer.CreateStep("remove-product-options-from-product",
	func(ctx composer.StepExecutionContext, pairs []services.ProductOptionProductPair) (composer.StepResponse[any], error) {
		if len(pairs) == 0 {
			return composer.NewStepResponse[any](nil, any(removedOptions{})), nil
		}
		svc := ProductService(ctx.Container)
		var productIDs []any
		seen := map[string]bool{}
		for _, p := range pairs {
			if !seen[p.ProductID] {
				seen[p.ProductID] = true
				productIDs = append(productIDs, p.ProductID)
			}
		}
		f := dal.NewRecord()
		f.Set("id", productIDs)
		products, err := svc.ListBySuffix(ctx.Context, "Products", f, utilsmodulessdk.FindConfig{Relations: []string{"options.values"}})
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		values := map[string][]any{}
		for _, p := range products {
			pid, _ := p.Get("id")
			opts, _ := p.Get("options")
			for _, o := range anyList(opts) {
				or, _ := o.(*dal.Record)
				if or == nil {
					continue
				}
				oid, _ := or.Get("id")
				vs, _ := or.Get("values")
				ids := []any{}
				for _, v := range anyList(vs) {
					if vr, ok := v.(*dal.Record); ok {
						id, _ := vr.Get("id")
						ids = append(ids, id)
					}
				}
				values[fmt.Sprint(pid)+"_"+fmt.Sprint(oid)] = ids
			}
		}
		var links []services.ProductOptionProductPair
		for _, p := range pairs {
			ids, ok := values[p.ProductID+"_"+p.ProductOptionID]
			if !ok {
				continue
			}
			l := p
			l.ValueIDs, l.HasValueIDs = ids, true
			links = append(links, l)
		}
		deleted, err := svc.RemoveProductOptionFromProduct(ctx.Context, pairs)
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil, any(removedOptions{links: links, deleted: deleted})), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		r, ok := data.(removedOptions)
		if !ok || (len(r.links) == 0 && len(r.deleted) == 0) {
			return nil
		}
		svc := ProductService(ctx.Container)
		if len(r.deleted) > 0 {
			if _, err := svc.Internal("ProductOption").Restore(ctx.Context, r.deleted); err != nil {
				return err
			}
		}
		if len(r.links) > 0 {
			return svc.AddProductOptionToProduct(ctx.Context, r.links)
		}
		return nil
	})

// UpdateProductOptionValuesOnProductStep mirrors
// updateProductOptionValuesOnProductStep (the compensation re-applies the
// inverse of the effective changes).
var UpdateProductOptionValuesOnProductStep = composer.CreateStep("update-product-option-values-on-product",
	func(ctx composer.StepExecutionContext, updates []services.ProductOptionProductValueUpdate) (composer.StepResponse[any], error) {
		var effective []services.ProductOptionProductValueUpdate
		for _, u := range updates {
			if len(u.Add) > 0 || len(u.Remove) > 0 {
				effective = append(effective, u)
			}
		}
		if len(effective) == 0 {
			return composer.NewStepResponse[any](nil, any([]services.ProductOptionProductValueUpdate{})), nil
		}
		svc := ProductService(ctx.Container)
		snapshot := func() (map[string]map[string]bool, error) {
			var ids []any
			seen := map[string]bool{}
			for _, u := range effective {
				if !seen[u.ProductID] {
					seen[u.ProductID] = true
					ids = append(ids, u.ProductID)
				}
			}
			f := dal.NewRecord()
			f.Set("id", ids)
			products, err := svc.ListBySuffix(ctx.Context, "Products", f, utilsmodulessdk.FindConfig{Relations: []string{"options.values"}})
			if err != nil {
				return nil, err
			}
			out := map[string]map[string]bool{}
			for _, p := range products {
				pid, _ := p.Get("id")
				opts, _ := p.Get("options")
				for _, o := range anyList(opts) {
					or, _ := o.(*dal.Record)
					if or == nil {
						continue
					}
					oid, _ := or.Get("id")
					set := map[string]bool{}
					vs, _ := or.Get("values")
					for _, v := range anyList(vs) {
						if vr, ok := v.(*dal.Record); ok {
							id, _ := vr.Get("id")
							set[fmt.Sprint(id)] = true
						}
					}
					out[fmt.Sprint(pid)+"_"+fmt.Sprint(oid)] = set
				}
			}
			return out, nil
		}
		before, err := snapshot()
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		if err := svc.UpdateProductOptionValuesOnProduct(ctx.Context, effective); err != nil {
			return composer.StepResponse[any]{}, err
		}
		after, err := snapshot()
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		var comp []services.ProductOptionProductValueUpdate
		for _, u := range effective {
			key := u.ProductID + "_" + u.ProductOptionID
			var adds, removes []any
			for id := range after[key] {
				if !before[key][id] {
					removes = append(removes, id)
				}
			}
			for id := range before[key] {
				if !after[key][id] {
					adds = append(adds, id)
				}
			}
			if len(adds) > 0 || len(removes) > 0 {
				comp = append(comp, services.ProductOptionProductValueUpdate{ProductID: u.ProductID, ProductOptionID: u.ProductOptionID, Add: adds, Remove: removes})
			}
		}
		return composer.NewStepResponse[any](nil, any(comp)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		comp, _ := data.([]services.ProductOptionProductValueUpdate)
		if len(comp) == 0 {
			return nil
		}
		return ProductService(ctx.Container).UpdateProductOptionValuesOnProduct(ctx.Context, comp)
	})
