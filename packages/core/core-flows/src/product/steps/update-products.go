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

// UpdateProductsStep mirrors updateProductsStep. The compensation reverts
// the scalar fields with upsertProducts (option link reconciliation is not
// ported yet).
var UpdateProductsStep = composer.CreateStep("update-products",
	func(ctx composer.StepExecutionContext, in UpdateProductsStepInput) (composer.StepResponse[[]*dal.Record], error) {
		svc := ProductService(ctx.Container)
		if in.Batch {
			for _, p := range in.Products {
				if id, _ := p.Get("id"); id == nil || id == "" {
					return composer.StepResponse[[]*dal.Record]{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Product ID is required when doing a batch update of products")
				}
			}
			if len(in.Products) == 0 {
				return composer.NewStepResponse([]*dal.Record{}, any([]*dal.Record{})), nil
			}
			idList := make([]any, len(in.Products))
			for i, p := range in.Products {
				idList[i], _ = p.Get("id")
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
			return composer.NewStepResponse(products, any(prev)), nil
		}
		var objs []common.ObjectLike
		if in.Update != nil {
			objs = []common.ObjectLike{in.Update}
		}
		selects, relations := common.GetSelectsAndRelationsFromObjectArray(objs, common.SelectsAndRelationsOptions{}, "")
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
		products, err := svc.UpdateProducts(ctx.Context, in.Selector, in.Update)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(products, any(prev)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		prev, _ := data.([]*dal.Record)
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
