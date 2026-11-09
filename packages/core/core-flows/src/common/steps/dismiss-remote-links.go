package steps

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// DismissRemoteLinkStep mirrors dismissRemoteLinkStep: the links are listed
// first (with their extra data) so the compensation can recreate them.
var DismissRemoteLinkStep = composer.CreateStep("dismiss-remote-links",
	func(ctx composer.StepExecutionContext, entries []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		if len(entries) == 0 {
			return composer.NewStepResponse([]*dal.Record{}, any([]*dal.Record{})), nil
		}
		link := LinkOf(ctx.Container)
		before, err := link.List(ctx.Context, entries, true)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		if _, err := link.Dismiss(ctx.Context, entries); err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(entries, any(before)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		links, _ := data.([]*dal.Record)
		if len(links) == 0 {
			return nil
		}
		_, err := LinkOf(ctx.Container).Create(ctx.Context, links)
		return err
	})
