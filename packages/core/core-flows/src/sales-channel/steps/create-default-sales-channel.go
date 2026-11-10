package steps

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

type defaultSalesChannelCompensation struct {
	ID           any
	ShouldDelete bool
}

// CreateDefaultSalesChannelStep mirrors createDefaultSalesChannelStep: the
// first sales channel, created from data when there is none.
var CreateDefaultSalesChannelStep = composer.CreateStep("create-default-sales-channel",
	func(ctx composer.StepExecutionContext, data *dal.Record) (composer.StepResponse[*dal.Record], error) {
		svc := salesChannels(ctx.Container)
		if svc == nil {
			return composer.NewStepResponse[*dal.Record](nil), nil
		}
		take := 1
		list, err := svc.Internal("SalesChannel").List(ctx.Context, dal.NewRecord(), utilsmodulessdk.FindConfig{Take: &take})
		if err != nil {
			return composer.StepResponse[*dal.Record]{}, err
		}
		shouldDelete := false
		var channel *dal.Record
		if len(list) > 0 {
			channel = list[0]
		} else {
			created, err := svc.Internal("SalesChannel").Create(ctx.Context, []*dal.Record{data})
			if err != nil {
				return composer.StepResponse[*dal.Record]{}, err
			}
			channel = created[0]
			shouldDelete = true
		}
		id, _ := channel.Get("id")
		return composer.NewStepResponse(channel, any(defaultSalesChannelCompensation{ID: id, ShouldDelete: shouldDelete})), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		c, ok := data.(defaultSalesChannelCompensation)
		if !ok || c.ID == nil || !c.ShouldDelete {
			return nil
		}
		_, err := salesChannels(ctx.Container).Internal("SalesChannel").Delete(ctx.Context, []any{c.ID})
		return err
	})
