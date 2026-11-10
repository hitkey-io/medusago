package steps

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

func cloneRecord(r *dal.Record) *dal.Record {
	out := dal.NewRecord()
	for _, k := range r.Keys() {
		v, _ := r.Get(k)
		out.Set(k, v)
	}
	return out
}

// ProcessProductOptionsForImportStep mirrors processProductOptionsForImportStep:
// options without id are created (exclusive by default) and every product's
// options become option_ids.
var ProcessProductOptionsForImportStep = composer.CreateStep("process-product-options-for-import",
	func(ctx composer.StepExecutionContext, products []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		type target struct{ product, option int }
		var toCreate []*dal.Record
		var targets []target
		for pi, p := range products {
			opts, _ := p.Get("options")
			for oi, o := range recordsOf(opts) {
				if id, _ := o.Get("id"); id != nil && id != "" {
					continue
				}
				c := cloneRecord(o)
				c.Delete("id")
				if ex, ok := c.Get("is_exclusive"); !ok || ex == nil {
					c.Set("is_exclusive", true)
				}
				toCreate = append(toCreate, c)
				targets = append(targets, target{pi, oi})
			}
		}
		var created []*dal.Record
		if len(toCreate) > 0 {
			var err error
			created, err = ProductService(ctx.Container).CreateProductOptions(ctx.Context, toCreate)
			if err != nil {
				return composer.StepResponse[[]*dal.Record]{}, err
			}
		}
		createdIDs := make([]any, len(created))
		byCoord := map[target]any{}
		for i, o := range created {
			id, _ := o.Get("id")
			createdIDs[i] = id
			byCoord[targets[i]] = id
		}
		out := make([]*dal.Record, len(products))
		for pi, p := range products {
			opts, _ := p.Get("options")
			var ids []any
			for oi, o := range recordsOf(opts) {
				id, _ := o.Get("id")
				if id == nil || id == "" {
					id = byCoord[target{pi, oi}]
				}
				if id != nil && id != "" {
					ids = append(ids, id)
				}
			}
			if len(ids) > 0 {
				c := cloneRecord(p)
				c.Delete("options")
				c.Set("option_ids", ids)
				out[pi] = c
			} else {
				out[pi] = p
			}
		}
		return composer.NewStepResponse(out, any(createdIDs)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		ids, _ := data.([]any)
		if len(ids) == 0 {
			return nil
		}
		_, err := ProductService(ctx.Container).Internal("ProductOption").Delete(ctx.Context, ids)
		return err
	})

func recordsOf(v any) []*dal.Record {
	var out []*dal.Record
	switch t := v.(type) {
	case []any:
		for _, x := range t {
			if r, ok := x.(*dal.Record); ok {
				out = append(out, r)
			}
		}
	case []*dal.Record:
		out = t
	}
	return out
}
