// Package steps mirrors @medusajs/core-flows/src/sales-channel/steps.
package steps

import (
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	modulessdk "github.com/hitkey-io/medusago/packages/core/modules-sdk/src"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// ProductChannelLink is { sales_channel_id, product_id }.
type ProductChannelLink struct {
	SalesChannelID string
	ProductID      string
}

// AssociateProductsWithSalesChannelsStep mirrors
// associateProductsWithSalesChannelsStep.
var AssociateProductsWithSalesChannelsStep = composer.CreateStep("associate-products-with-channels",
	func(ctx composer.StepExecutionContext, links []ProductChannelLink) (composer.StepResponse[[]*dal.Record], error) {
		if len(links) == 0 {
			return composer.NewStepResponse([]*dal.Record{}, any([]*dal.Record{})), nil
		}
		defs := make([]*dal.Record, len(links))
		for i, l := range links {
			defs[i] = modulessdk.NewLinkDefinition("product", "product_id", l.ProductID, "sales_channel", "sales_channel_id", l.SalesChannelID)
		}
		created, err := commonsteps.LinkOf(ctx.Container).Create(ctx.Context, defs)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(created, any(defs)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		links, _ := data.([]*dal.Record)
		if links == nil {
			return nil
		}
		_, err := commonsteps.LinkOf(ctx.Container).Dismiss(ctx.Context, links)
		return err
	})
