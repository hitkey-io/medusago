// Package steps mirrors @medusajs/core-flows/src/defaults/steps.
package steps

import (
	storesteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/store/steps"
	storeworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/store/workflows"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

type defaultStoreCompensation struct {
	StoreID      any
	ShouldDelete bool
}

// CreateDefaultStoreStep mirrors createDefaultStoreStep: the first store,
// created with EUR as the default currency when there is none.
var CreateDefaultStoreStep = composer.CreateStep("create-default-store",
	func(ctx composer.StepExecutionContext, store *dal.Record) (composer.StepResponse[*dal.Record], error) {
		svc := storesteps.StoreService(ctx.Container)
		if svc == nil {
			return composer.NewStepResponse[*dal.Record](nil), nil
		}
		take := 1
		list, err := svc.ListBySuffix(ctx.Context, "Stores", dal.NewRecord(), utilsmodulessdk.FindConfig{Take: &take})
		if err != nil {
			return composer.StepResponse[*dal.Record]{}, err
		}
		shouldDelete := false
		var out *dal.Record
		if len(list) > 0 {
			out = list[0]
		} else {
			in := dal.NewRecord()
			for _, k := range store.Keys() {
				v, _ := store.Get(k)
				in.Set(k, v)
			}
			eur := dal.NewRecord()
			eur.Set("currency_code", "eur")
			eur.Set("is_default", true)
			in.Set("supported_currencies", []any{eur})
			created, err := storeworkflows.CreateStoresWorkflow.Run(ctx.Context, ctx.Container, storeworkflows.CreateStoresWorkflowInput{Stores: []*dal.Record{in}})
			if err != nil {
				return composer.StepResponse[*dal.Record]{}, err
			}
			out = created[0]
			shouldDelete = true
		}
		id, _ := out.Get("id")
		return composer.NewStepResponse(out, any(defaultStoreCompensation{StoreID: id, ShouldDelete: shouldDelete})), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		c, ok := data.(defaultStoreCompensation)
		if !ok || !c.ShouldDelete {
			return nil
		}
		_, err := storesteps.StoreService(ctx.Container).Internal("Store").Delete(ctx.Context, []any{c.StoreID})
		return err
	})
