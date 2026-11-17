package steps

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// UpdateProductsStepInput mirrors UpdateProductsStepInput: either
// Selector/Update or Products.
type UpdateProductsStepInput struct {
	Selector *dal.Record
	Update   *dal.Record
	Products []*dal.Record
	Batch    bool
}

// buildRestoreData mirrors buildRestoreData: only the fields the update sets
// are restored, so unrelated references loaded with the product (e.g. type
// next to type_id) don't override the restored values. Metadata is merged on
// update, so the keys the update added are cleared with "".
func buildRestoreData(product, update *dal.Record) *dal.Record {
	restore := dal.NewRecord()
	restore.Set("id", func() any { v, _ := product.Get("id"); return v }())
	options, _ := product.Get("options")
	restore.Set("options", options)
	if update != nil {
		for _, key := range update.Keys() {
			if key == "options" || !product.Has(key) {
				continue
			}
			v, _ := product.Get(key)
			restore.Set(key, v)
		}
	}
	metadata, _ := restore.Get("metadata")
	current, isRec := metadata.(*dal.Record)
	var updateMetadata *dal.Record
	if update != nil {
		updateMetadata, _ = func() any { v, _ := update.Get("metadata"); return v }().(*dal.Record)
	}
	if isRec && current != nil && current.Len() > 0 && updateMetadata != nil {
		merged := dal.NewRecord()
		for _, k := range current.Keys() {
			v, _ := current.Get(k)
			merged.Set(k, v)
		}
		for _, k := range updateMetadata.Keys() {
			if !merged.Has(k) {
				merged.Set(k, "")
			}
		}
		restore.Set("metadata", merged)
	}
	return restore
}

// updateProductsCompensation is the compensation data of updateProductsStep.
func updateProductsCompensation(prev []*dal.Record, targetOptionIDs *dal.Record) *dal.Record {
	data := dal.NewRecord()
	list := make([]any, len(prev))
	for i, p := range prev {
		list[i] = p
	}
	data.Set("prevProducts", list)
	data.Set("targetOptionIdsByProductId", targetOptionIDs)
	return data
}

// UpdateProductsStep mirrors updateProductsStep. The compensation reverts
// the scalar fields with upsertProducts (option link reconciliation is not
// ported yet).
var UpdateProductsStep = composer.CreateStep("update-products",
	func(ctx composer.StepExecutionContext, in UpdateProductsStepInput) (composer.StepResponse[[]*dal.Record], error) {
		svc := ProductService(ctx.Container)
		targetOptionIDs := dal.NewRecord()
		if in.Batch {
			for _, p := range in.Products {
				if id, _ := p.Get("id"); id == nil || id == "" {
					return composer.StepResponse[[]*dal.Record]{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Product ID is required when doing a batch update of products")
				}
			}
			if len(in.Products) == 0 {
				return composer.NewStepResponse([]*dal.Record{}, any(updateProductsCompensation(nil, targetOptionIDs))), nil
			}
			idList := make([]any, len(in.Products))
			updateByID := map[any]*dal.Record{}
			for i, p := range in.Products {
				idList[i], _ = p.Get("id")
				// new Map(entries): the last entry of an id wins
				updateByID[idList[i]] = p
				if ids, ok := p.Get("option_ids"); ok {
					if s, isStr := idList[i].(string); isStr {
						targetOptionIDs.Set(s, ids)
					}
				}
			}
			f := dal.NewRecord()
			f.Set("id", idList)
			prev, err := svc.ListBySuffix(ctx.Context, "Products", f, utilsmodulessdk.FindConfig{Relations: []string{"options.values"}})
			if err != nil {
				return composer.StepResponse[[]*dal.Record]{}, err
			}
			products, err := svc.UpsertProducts(ctx.Context, in.Products)
			if err != nil {
				return composer.StepResponse[[]*dal.Record]{}, err
			}
			restore := make([]*dal.Record, len(prev))
			for i, p := range prev {
				id, _ := p.Get("id")
				restore[i] = buildRestoreData(p, updateByID[id])
			}
			return composer.NewStepResponse(products, any(updateProductsCompensation(restore, targetOptionIDs))), nil
		}
		var objs []common.ObjectLike
		if in.Update != nil {
			objs = []common.ObjectLike{in.Update}
		}
		selects, relations := common.GetSelectsAndRelationsFromObjectArray(objs, common.SelectsAndRelationsOptions{ObjectFields: []string{"metadata"}}, "")
		hasOV := false
		for _, r := range relations {
			if r == "options.values" {
				hasOV = true
			}
		}
		if !hasOV {
			relations = append(relations, "options.values")
		}
		prev, err := svc.ListBySuffix(ctx.Context, "Products", in.Selector, utilsmodulessdk.FindConfig{Select: selects, Relations: relations})
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		if in.Update != nil {
			if ids, ok := in.Update.Get("option_ids"); ok {
				for _, p := range prev {
					id, _ := p.Get("id")
					if s, ok := id.(string); ok {
						targetOptionIDs.Set(s, ids)
					}
				}
			}
		}
		products, err := svc.UpdateProducts(ctx.Context, in.Selector, in.Update)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		restore := make([]*dal.Record, len(prev))
		for i, p := range prev {
			restore[i] = buildRestoreData(p, in.Update)
		}
		return composer.NewStepResponse(products, any(updateProductsCompensation(restore, targetOptionIDs))), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		var prev []*dal.Record
		if r, ok := data.(*dal.Record); ok && r != nil {
			list, _ := r.Get("prevProducts")
			switch t := list.(type) {
			case []any:
				for _, x := range t {
					if p, ok := x.(*dal.Record); ok {
						prev = append(prev, p)
					}
				}
			case []*dal.Record:
				prev = t
			}
		}
		if len(prev) == 0 {
			return nil
		}
		rest := make([]*dal.Record, len(prev))
		for i, p := range prev {
			r := dal.NewRecord()
			for _, k := range p.Keys() {
				if k == "options" {
					continue
				}
				v, _ := p.Get(k)
				r.Set(k, v)
			}
			rest[i] = r
		}
		_, err := ProductService(ctx.Container).UpsertProducts(ctx.Context, rest)
		return err
	})
