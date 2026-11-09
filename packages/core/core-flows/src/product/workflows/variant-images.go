package workflows

import (
	"fmt"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// BatchImagesInput mirrors { image_id | variant_id, add?, remove? }.
type BatchImagesInput struct {
	ID     any
	Add    []any
	Remove []any
}

// BatchImagesResult mirrors { added, removed }.
type BatchImagesResult struct {
	Added   []any
	Removed []any
}

func nonNilIDs(l []any) []any {
	if l == nil {
		return []any{}
	}
	return l
}

func queryRows(e *composer.Execution, name, entity string, fields []string, filters *dal.Record, isList bool) (any, error) {
	res, err := commonsteps.UseQueryGraphStep.Config(name, composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
		GraphInput: query.GraphInput{Entity: entity, Fields: fields, Filters: filters}, IsList: &isList,
	})
	if err != nil {
		return nil, err
	}
	return res.Data, nil
}

func field(v any, k string) any {
	r, _ := v.(*dal.Record)
	if r == nil {
		return nil
	}
	x, _ := r.Get(k)
	return x
}

func idEventsOf(ids []any) []any {
	out := make([]any, len(ids))
	for i, id := range ids {
		ev := dal.NewRecord()
		ev.Set("id", id)
		out[i] = ev
	}
	return out
}

// BatchImageVariantsWorkflow mirrors batchImageVariantsWorkflow.
var BatchImageVariantsWorkflow = composer.CreateWorkflow("batch-image-variants",
	func(e *composer.Execution, in BatchImagesInput) (BatchImagesResult, error) {
		add, remove := nonNilIDs(in.Add), nonNilIDs(in.Remove)
		var out BatchImagesResult
		err := composer.Parallelize(
			func() (err error) {
				out.Added, err = steps.AddImageToVariantsStep.Run(e, steps.VariantImagesInput{ImageIDs: []any{in.ID}, VariantIDs: add, IDs: add})
				return err
			},
			func() (err error) {
				out.Removed, err = steps.RemoveImageFromVariantsStep.Run(e, steps.VariantImagesInput{ImageIDs: []any{in.ID}, VariantIDs: remove, IDs: remove})
				return err
			},
		)
		if err != nil {
			return BatchImagesResult{}, err
		}
		if len(remove) > 0 {
			vf := dal.NewRecord()
			vf.Set("id", remove)
			if _, err := queryRows(e, "get-variants-for-thumbnail-check", "variants", []string{"id", "thumbnail"}, vf, true); err != nil {
				return BatchImagesResult{}, err
			}
			imf := dal.NewRecord()
			imf.Set("id", in.ID)
			image, err := queryRows(e, "get-image-for-thumbnail-check", "product_image", []string{"id", "url"}, imf, false)
			if err != nil {
				return BatchImagesResult{}, err
			}
			if url, ok := field(image, "url").(string); ok {
				sel := dal.NewRecord()
				sel.Set("id", remove)
				sel.Set("thumbnail", url)
				upd := dal.NewRecord()
				upd.Set("thumbnail", nil)
				if _, err := steps.UpdateProductVariantsStep.Run(e, steps.UpdateProductVariantsStepInput{Selector: sel, Update: upd}); err != nil {
					return BatchImagesResult{}, err
				}
			}
		}
		var ids []any
		seen := map[string]bool{}
		for _, id := range append(append([]any{}, add...), remove...) {
			k := fmt.Sprint(id)
			if !seen[k] {
				seen[k] = true
				ids = append(ids, id)
			}
		}
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "product-variant.updated", Data: idEventsOf(ids)}); err != nil {
			return BatchImagesResult{}, err
		}
		return out, nil
	})

// BatchVariantImagesWorkflow mirrors batchVariantImagesWorkflow.
var BatchVariantImagesWorkflow = composer.CreateWorkflow("batch-variant-images",
	func(e *composer.Execution, in BatchImagesInput) (BatchImagesResult, error) {
		add, remove := nonNilIDs(in.Add), nonNilIDs(in.Remove)
		var out BatchImagesResult
		err := composer.Parallelize(
			func() (err error) {
				out.Added, err = steps.AddImagesToVariantStep.Run(e, steps.VariantImagesInput{ImageIDs: add, VariantIDs: []any{in.ID}, IDs: add})
				return err
			},
			func() (err error) {
				out.Removed, err = steps.RemoveImagesFromVariantStep.Run(e, steps.VariantImagesInput{ImageIDs: remove, VariantIDs: []any{in.ID}, IDs: remove})
				return err
			},
		)
		if err != nil {
			return BatchImagesResult{}, err
		}
		if len(remove) > 0 {
			vf := dal.NewRecord()
			vf.Set("id", in.ID)
			variant, err := queryRows(e, "get-variant-thumbnail", "variant", []string{"id", "thumbnail"}, vf, false)
			if err != nil {
				return BatchImagesResult{}, err
			}
			imf := dal.NewRecord()
			imf.Set("id", remove)
			images, err := queryRows(e, "get-removed-images", "product_image", []string{"id", "url"}, imf, true)
			if err != nil {
				return BatchImagesResult{}, err
			}
			thumb := field(variant, "thumbnail")
			match := false
			list, _ := images.([]any)
			for _, im := range list {
				if field(im, "url") == thumb {
					match = true
				}
			}
			if match {
				sel := dal.NewRecord()
				sel.Set("id", in.ID)
				upd := dal.NewRecord()
				upd.Set("thumbnail", nil)
				if _, err := steps.UpdateProductVariantsStep.Run(e, steps.UpdateProductVariantsStepInput{Selector: sel, Update: upd}); err != nil {
					return BatchImagesResult{}, err
				}
			}
		}
		events := []any{}
		if len(add) > 0 || len(remove) > 0 {
			events = idEventsOf([]any{in.ID})
		}
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "product-variant.updated", Data: events}); err != nil {
			return BatchImagesResult{}, err
		}
		return out, nil
	})
