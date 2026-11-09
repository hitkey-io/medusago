package steps

import (
	modulessdk "github.com/hitkey-io/medusago/packages/core/modules-sdk/src"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// LinkOf resolves Link from a container.
func LinkOf(c composer.Container) *modulessdk.Link {
	l, _ := c.Resolve(common.ContainerRegistrationKeys.LINK).(*modulessdk.Link)
	return l
}

// CreateRemoteLinkStep mirrors createRemoteLinkStep.
var CreateRemoteLinkStep = composer.CreateStep("create-remote-links",
	func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		if len(data) == 0 {
			return composer.NewStepResponse([]*dal.Record{}, any([]*dal.Record{})), nil
		}
		if _, err := LinkOf(ctx.Container).Create(ctx.Context, data); err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(data, any(data)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		links, _ := data.([]*dal.Record)
		if links == nil {
			return nil
		}
		_, err := LinkOf(ctx.Container).Dismiss(ctx.Context, links)
		return err
	})
