package steps

import (
	"context"

	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// GetTranslatedTaxLinesStepInput mirrors GetTranslatedTaxLinesStepInput.
type GetTranslatedTaxLinesStepInput struct {
	ItemTaxLines     []*dal.Record
	ShippingTaxLines []*dal.Record
	Locale           string
}

// TranslatedTaxLines is { itemTaxLines, shippingTaxLines }.
type TranslatedTaxLines struct {
	ItemTaxLines     []*dal.Record
	ShippingTaxLines []*dal.Record
}

// ApplyTranslationsToTaxLines mirrors applyTranslationsToTaxLines: the name
// of the tax lines with a rate is the (translated) name of their rate. The
// tax lines are copied.
func ApplyTranslationsToTaxLines(ctx context.Context, q *query.Query, taxLines []*dal.Record, locale string) ([]*dal.Record, error) {
	rates := make([]any, len(taxLines))
	for i, l := range taxLines {
		r := dal.NewRecord()
		id, _ := l.Get("rate_id")
		name, _ := l.Get("name")
		r.Set("id", id)
		r.Set("name", name)
		rates[i] = r
	}
	if err := q.ApplyTranslations(ctx, locale, rates); err != nil {
		return nil, err
	}
	names := map[any]any{}
	for _, x := range rates {
		r := x.(*dal.Record)
		id, _ := r.Get("id")
		if id == nil || id == "" {
			continue
		}
		names[id], _ = r.Get("name")
	}
	out := make([]*dal.Record, len(taxLines))
	for i, l := range taxLines {
		c := dal.NewRecord()
		for _, k := range l.Keys() {
			v, _ := l.Get(k)
			c.Set(k, v)
		}
		if id, _ := l.Get("rate_id"); id != nil && id != "" {
			c.Set("name", names[id])
		}
		out[i] = c
	}
	return out, nil
}

// GetTranslatedTaxLinesStep mirrors getTranslatedTaxLinesStep.
var GetTranslatedTaxLinesStep = composer.CreateStep("get-translated-tax-lines-step",
	func(ctx composer.StepExecutionContext, in GetTranslatedTaxLinesStepInput) (composer.StepResponse[TranslatedTaxLines], error) {
		q, _ := ctx.Container.Resolve(common.ContainerRegistrationKeys.QUERY).(*query.Query)
		if !q.TranslationsEnabled() {
			return composer.NewStepResponse(TranslatedTaxLines{ItemTaxLines: in.ItemTaxLines, ShippingTaxLines: in.ShippingTaxLines}), nil
		}
		items, err := ApplyTranslationsToTaxLines(ctx.Context, q, in.ItemTaxLines, in.Locale)
		if err != nil {
			return composer.StepResponse[TranslatedTaxLines]{}, err
		}
		shipping, err := ApplyTranslationsToTaxLines(ctx.Context, q, in.ShippingTaxLines, in.Locale)
		if err != nil {
			return composer.StepResponse[TranslatedTaxLines]{}, err
		}
		return composer.NewStepResponse(TranslatedTaxLines{ItemTaxLines: items, ShippingTaxLines: shipping}), nil
	}, nil)
