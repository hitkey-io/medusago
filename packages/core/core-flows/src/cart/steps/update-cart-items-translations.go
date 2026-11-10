package steps

import (
	"context"

	. "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/utils"
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// UpdateCartItemsTranslationsStepInput mirrors
// UpdateCartItemsTranslationsStepInput.
type UpdateCartItemsTranslationsStepInput struct {
	CartID string
	Locale string
	// Items are pre-loaded items (nil: they are read in batches).
	Items []*dal.Record
}

const cartItemsTranslationsBatchSize = 100

var cartItemsTranslationsLineItemFields = []string{"id", "variant_id", "product_id", "title", "subtitle", "product_title", "product_description", "product_subtitle", "product_type", "product_collection", "product_handle", "variant_title"}

var cartItemTranslationSnapshotKeys = []string{"id", "title", "subtitle", "product_title", "product_description", "product_subtitle", "product_type", "product_collection", "product_handle", "variant_title"}

func cartItemsTranslationsSnapshot(item *dal.Record) *dal.Record {
	r := dal.NewRecord()
	for _, k := range cartItemTranslationSnapshotKeys {
		if v, ok := item.Get(k); ok {
			r.Set(k, v)
		}
	}
	return r
}

func restoreCartItemsTranslations(ctx context.Context, c composer.Container, original []*dal.Record) error {
	for i := 0; i < len(original); i += cartItemsTranslationsBatchSize {
		end := i + cartItemsTranslationsBatchSize
		if end > len(original) {
			end = len(original)
		}
		batch := make([]any, 0, end-i)
		for _, r := range original[i:end] {
			batch = append(batch, r)
		}
		if _, err := CartOf(c).Internal("LineItem").Update(ctx, batch); err != nil {
			return err
		}
	}
	return nil
}

// UpdateCartItemsTranslationsStep mirrors updateCartItemsTranslationsStep:
// the line items are translated again in the new cart locale.
var UpdateCartItemsTranslationsStep = composer.CreateStep("update-cart-items-translations",
	func(ctx composer.StepExecutionContext, in UpdateCartItemsTranslationsStepInput) (composer.StepResponse[any], error) {
		var original []*dal.Record
		q, _ := ctx.Container.Resolve(common.ContainerRegistrationKeys.QUERY).(*query.Query)
		if !q.TranslationsEnabled() {
			return composer.NewStepResponse[any](nil, any([]*dal.Record{})), nil
		}
		processBatch := func(items []*dal.Record) error {
			var variantIDs []any
			seen := map[any]bool{}
			for _, item := range items {
				id := Get(item, "variant_id")
				if s, ok := id.(string); ok && s != "" && !seen[id] {
					seen[id] = true
					variantIDs = append(variantIDs, id)
				}
			}
			if len(variantIDs) == 0 {
				return nil
			}
			for _, item := range items {
				original = append(original, cartItemsTranslationsSnapshot(item))
			}
			f := dal.NewRecord()
			f.Set("id", variantIDs)
			res, err := q.Graph(ctx.Context, query.GraphInput{Entity: "variants", Filters: f, Fields: ProductVariantsFields}, joiner.Options{Locale: in.Locale})
			if err != nil {
				return err
			}
			variants := make([]*dal.Record, 0, len(res.Data))
			for _, v := range res.Data {
				if r, ok := v.(*dal.Record); ok {
					variants = append(variants, r)
				}
			}
			copies := make([]*dal.Record, len(items))
			for i, item := range items {
				copies[i] = Copy(item)
			}
			translated := commonsteps.ApplyTranslationsToItems(copies, variants)
			var updates []any
			for _, item := range translated {
				if Get(item, "id") == nil || Get(item, "id") == "" {
					continue
				}
				updates = append(updates, cartItemsTranslationsSnapshot(item))
			}
			if len(updates) > 0 {
				if _, err := CartOf(ctx.Container).Internal("LineItem").Update(ctx.Context, updates); err != nil {
					return err
				}
			}
			return nil
		}
		fail := func(err error) (composer.StepResponse[any], error) {
			_ = restoreCartItemsTranslations(ctx.Context, ctx.Container, original)
			return composer.StepResponse[any]{}, err
		}
		if len(in.Items) > 0 {
			if err := processBatch(in.Items); err != nil {
				return fail(err)
			}
			return composer.NewStepResponse[any](nil, any(original)), nil
		}
		for offset := 0; ; {
			f := dal.NewRecord()
			f.Set("cart_id", in.CartID)
			p := dal.NewRecord()
			p.Set("take", float64(cartItemsTranslationsBatchSize))
			p.Set("skip", float64(offset))
			res, err := q.Graph(ctx.Context, query.GraphInput{Entity: "line_items", Filters: f, Fields: cartItemsTranslationsLineItemFields, Pagination: p}, joiner.Options{})
			if err != nil {
				return fail(err)
			}
			if len(res.Data) == 0 {
				break
			}
			items := make([]*dal.Record, 0, len(res.Data))
			for _, x := range res.Data {
				if r, ok := x.(*dal.Record); ok {
					items = append(items, r)
				}
			}
			if err := processBatch(items); err != nil {
				return fail(err)
			}
			offset += len(res.Data)
			if len(res.Data) != cartItemsTranslationsBatchSize {
				break
			}
		}
		return composer.NewStepResponse[any](nil, any(original)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		original, _ := data.([]*dal.Record)
		if len(original) == 0 {
			return nil
		}
		return restoreCartItemsTranslations(ctx.Context, ctx.Container, original)
	})
