package steps

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// ImagePairs builds { image_id, variant_id } pairs.
func ImagePairs(imageIDs, variantIDs []any) []*dal.Record {
	var out []*dal.Record
	for _, i := range imageIDs {
		for _, v := range variantIDs {
			r := dal.NewRecord()
			r.Set("image_id", i)
			r.Set("variant_id", v)
			out = append(out, r)
		}
	}
	return out
}

// VariantImagesInput is { image_id | variant_id, ids }.
type VariantImagesInput struct {
	ImageIDs   []any
	VariantIDs []any
	// IDs are the added or removed ids (variants or images).
	IDs []any
}

func imageStep(name string, add bool) *composer.Step[VariantImagesInput, []any] {
	apply := func(ctx composer.StepExecutionContext, pairs []*dal.Record, add bool) error {
		if add {
			_, err := ProductService(ctx.Container).AddImageToVariant(ctx.Context, pairs)
			return err
		}
		return ProductService(ctx.Container).RemoveImageFromVariant(ctx.Context, pairs)
	}
	return composer.CreateStep(name,
		func(ctx composer.StepExecutionContext, in VariantImagesInput) (composer.StepResponse[[]any], error) {
			if len(in.IDs) == 0 {
				return composer.NewStepResponse([]any{}, any(VariantImagesInput{})), nil
			}
			if err := apply(ctx, ImagePairs(in.ImageIDs, in.VariantIDs), add); err != nil {
				return composer.StepResponse[[]any]{}, err
			}
			return composer.NewStepResponse(in.IDs, any(in)), nil
		},
		func(ctx composer.StepExecutionContext, data any) error {
			in, ok := data.(VariantImagesInput)
			if !ok || len(in.IDs) == 0 {
				return nil
			}
			return apply(ctx, ImagePairs(in.ImageIDs, in.VariantIDs), !add)
		})
}

// Variant image steps.
var (
	AddImageToVariantsStep      = imageStep("add-image-to-variants", true)
	RemoveImageFromVariantsStep = imageStep("remove-image-from-variants", false)
	AddImagesToVariantStep      = imageStep("add-images-to-variant", true)
	RemoveImagesFromVariantStep = imageStep("remove-images-from-variant", false)
)
