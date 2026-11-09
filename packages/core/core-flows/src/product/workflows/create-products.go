package workflows

import (
	"fmt"
	"strings"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/steps"
	saleschannelsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/sales-channel/steps"
	modulessdk "github.com/hitkey-io/medusago/packages/core/modules-sdk/src"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// ValidateProductInputStep mirrors validateProductInputStep.
var ValidateProductInputStep = composer.CreateStep("validate-product-input",
	func(_ composer.StepExecutionContext, products []*dal.Record) (composer.StepResponse[any], error) {
		var missing []string
		for _, p := range products {
			if len(records(get(p, "options"))) == 0 {
				missing = append(missing, fmt.Sprint(get(p, "title")))
			}
		}
		if len(missing) > 0 {
			return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Product options are not provided for: [%s].", strings.Join(missing, ", ")))
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// CreateProductsInput mirrors CreateProductsWorkflowInput.
type CreateProductsInput struct {
	Products       []*dal.Record
	AdditionalData *dal.Record
}

// CreateProductsWorkflow mirrors createProductsWorkflow.
var CreateProductsWorkflow = composer.CreateWorkflow("create-products",
	func(e *composer.Execution, input CreateProductsInput) ([]*dal.Record, error) {
		withoutExternal := make([]*dal.Record, len(input.Products))
		for i, p := range input.Products {
			withoutExternal[i] = without(p, "sales_channels", "shipping_profile_id", "variants")
		}
		if _, err := ValidateProductInputStep.Run(e, input.Products); err != nil {
			return nil, err
		}
		created, err := steps.CreateProductsStep.Run(e, withoutExternal)
		if err != nil {
			return nil, err
		}
		var channelLinks []saleschannelsteps.ProductChannelLink
		for i, p := range created {
			for _, sc := range records(get(input.Products[i], "sales_channels")) {
				channelLinks = append(channelLinks, saleschannelsteps.ProductChannelLink{SalesChannelID: fmt.Sprint(get(sc, "id")), ProductID: fmt.Sprint(get(p, "id"))})
			}
		}
		if _, err := saleschannelsteps.AssociateProductsWithSalesChannelsStep.Run(e, channelLinks); err != nil {
			return nil, err
		}
		var profileLinks []*dal.Record
		for i, p := range created {
			if sp := get(input.Products[i], "shipping_profile_id"); sp != nil && sp != "" {
				profileLinks = append(profileLinks, modulessdk.NewLinkDefinition("product", "product_id", get(p, "id"), "fulfillment", "shipping_profile_id", sp))
			}
		}
		if _, err := commonsteps.CreateRemoteLinkStep.Run(e, profileLinks); err != nil {
			return nil, err
		}
		var variants []*dal.Record
		for i, p := range created {
			for _, v := range records(get(input.Products[i], "variants")) {
				r := dal.NewRecord()
				r.Set("product_id", get(p, "id"))
				for _, k := range v.Keys() {
					x, _ := v.Get(k)
					r.Set(k, x)
				}
				variants = append(variants, r)
			}
		}
		createdVariants, err := CreateProductVariantsWorkflow.RunAsStep(e, CreateProductVariantsInput{ProductVariants: variants})
		if err != nil {
			return nil, err
		}
		byProduct := map[any][]*dal.Record{}
		for _, v := range createdVariants {
			pid := get(v, "product_id")
			byProduct[pid] = append(byProduct[pid], v)
		}
		for _, p := range created {
			vs := byProduct[get(p, "id")]
			if vs == nil {
				vs = []*dal.Record{}
			}
			p.Set("variants", vs)
		}
		events := make([]any, len(created))
		for i, p := range created {
			ev := dal.NewRecord()
			ev.Set("id", get(p, "id"))
			events[i] = ev
		}
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "product.created", Data: events}); err != nil {
			return nil, err
		}
		hookInput := dal.NewRecord()
		hookInput.Set("products", created)
		hookInput.Set("additional_data", input.AdditionalData)
		if err := createProductsHook(e, hookInput); err != nil {
			return nil, err
		}
		return created, nil
	})

var createProductsHook = func(e *composer.Execution, input *dal.Record) error { return nil }

func init() {
	createProductsHook = func(e *composer.Execution, input *dal.Record) error {
		return CreateProductsWorkflow.CreateHook(e, "productsCreated", input)
	}
}

// idEvents mirrors rows.map((v) => ({ id: v.id })).
func idEvents(rows []*dal.Record) []any {
	out := make([]any, len(rows))
	for i, r := range rows {
		ev := dal.NewRecord()
		id, _ := r.Get("id")
		ev.Set("id", id)
		out[i] = ev
	}
	return out
}

// deleteInput builds { [module]: { [key]: ids } }.
func deleteInput(module, key string, ids []any) *dal.Record {
	inner := dal.NewRecord()
	inner.Set(key, ids)
	out := dal.NewRecord()
	out.Set(module, inner)
	return out
}
