package steps

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// DismissRemoteLinkStep mirrors dismissRemoteLinkStep. The compensation
// recreates the dismissed links (upstream lists them first to keep extra
// columns; link.list is not ported yet, so extra columns are not restored).
var DismissRemoteLinkStep = composer.CreateStep("dismiss-remote-links",
	func(ctx composer.StepExecutionContext, entries []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		if len(entries) == 0 {
			return composer.NewStepResponse([]*dal.Record{}, any([]*dal.Record{})), nil
		}
		if _, err := LinkOf(ctx.Container).Dismiss(ctx.Context, entries); err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(entries, any(entries)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		links, _ := data.([]*dal.Record)
		if len(links) == 0 {
			return nil
		}
		_, err := LinkOf(ctx.Container).Create(ctx.Context, links)
		return err
	})
