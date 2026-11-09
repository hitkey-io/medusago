package steps

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// UpdateRemoteLinksStep mirrors updateRemoteLinksStep: every link must
// exist; link.create upserts their data.
var UpdateRemoteLinksStep = composer.CreateStep("update-remote-links-step",
	func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		if len(data) == 0 {
			return composer.NewStepResponse([]*dal.Record{}, any([]*dal.Record{})), nil
		}
		link := LinkOf(ctx.Container)
		before, err := link.List(ctx.Context, data, true)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		if len(before) != len(data) {
			return composer.StepResponse[[]*dal.Record]{}, common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, "Could not find all existing links from data")
		}
		if _, err := link.Create(ctx.Context, data); err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(data, any(before)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		before, _ := data.([]*dal.Record)
		if len(before) == 0 {
			return nil
		}
		_, err := LinkOf(ctx.Container).Create(ctx.Context, before)
		return err
	})
