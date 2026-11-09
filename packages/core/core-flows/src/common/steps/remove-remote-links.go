package steps

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// RemoveRemoteLinkStep mirrors removeRemoteLinkStep: the entries
// ({ [module]: { [linkableKey]: ids } }) are grouped and passed to
// link.delete; the compensation restores them.
var RemoveRemoteLinkStep = composer.CreateStep("remove-remote-links",
	func(ctx composer.StepExecutionContext, entries []*dal.Record) (composer.StepResponse[*dal.Record], error) {
		if len(entries) == 0 {
			return composer.NewStepResponse[*dal.Record](nil), nil
		}
		grouped := dal.NewRecord()
		for _, entry := range entries {
			for _, module := range entry.Keys() {
				mv, _ := entry.Get(module)
				keys, _ := mv.(*dal.Record)
				cur, _ := grouped.Get(module)
				g, _ := cur.(*dal.Record)
				if g == nil {
					g = dal.NewRecord()
					grouped.Set(module, g)
				}
				if keys == nil {
					continue
				}
				for _, k := range keys.Keys() {
					v, _ := keys.Get(k)
					prev, _ := g.Get(k)
					list, _ := prev.([]any)
					if vs, ok := v.([]any); ok {
						list = append(list, vs...)
					} else if vs, ok := v.([]string); ok {
						for _, s := range vs {
							list = append(list, s)
						}
					} else {
						list = append(list, v)
					}
					g.Set(k, list)
				}
			}
		}
		LinkOf(ctx.Container).Delete(ctx.Context, grouped)
		return composer.NewStepResponse(grouped, any(grouped)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		grouped, _ := data.(*dal.Record)
		if grouped == nil {
			return nil
		}
		LinkOf(ctx.Container).Restore(ctx.Context, grouped)
		return nil
	})
