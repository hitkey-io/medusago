// Package workflows mirrors @medusajs/core-flows/src/api-key/workflows
// (steps included).
package workflows

import (
	"fmt"
	"strings"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	modulessdk "github.com/hitkey-io/medusago/packages/core/modules-sdk/src"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	"github.com/hitkey-io/medusago/packages/modules/api-key/src/services"
)

func apiKeys(c composer.Container) *services.ApiKeyModuleService {
	s, _ := c.Resolve(utilsmodulessdk.API_KEY).(*services.ApiKeyModuleService)
	return s
}

// CreateApiKeysStep mirrors createApiKeysStep.
var CreateApiKeysStep = composer.CreateStep("create-api-keys",
	func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		created, err := apiKeys(ctx.Container).CreateApiKeys(ctx.Context, data)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(created, any(commonsteps.RecordIDs(created))), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		ids, _ := data.([]any)
		if len(ids) == 0 {
			return nil
		}
		return apiKeys(ctx.Container).DeleteApiKeys(ctx.Context, ids)
	})

// CreateApiKeysWorkflow mirrors createApiKeysWorkflow.
var CreateApiKeysWorkflow = composer.CreateWorkflow("create-api-keys",
	func(e *composer.Execution, data []*dal.Record) ([]*dal.Record, error) {
		created, err := CreateApiKeysStep.Run(e, data)
		if err != nil {
			return nil, err
		}
		h := dal.NewRecord()
		h.Set("apiKeys", created)
		return created, composer.CreateHook(e, "apiKeysCreated", h)
	})

// SelectorInput is { selector, update|revoke }.
type SelectorInput struct {
	Selector *dal.Record
	Data     *dal.Record
}

// UpdateApiKeysStep mirrors updateApiKeysStep.
var UpdateApiKeysStep = composer.CreateStep("update-api-keys",
	func(ctx composer.StepExecutionContext, in SelectorInput) (composer.StepResponse[[]*dal.Record], error) {
		svc := apiKeys(ctx.Container)
		prev, err := commonsteps.PrevDataFor(ctx.Context, svc, "ApiKeys", in.Selector, in.Data)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		out, err := svc.UpdateApiKeys(ctx.Context, in.Selector, in.Data)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(out, any(prev)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		prev, _ := data.([]*dal.Record)
		if len(prev) == 0 {
			return nil
		}
		in := make([]*dal.Record, len(prev))
		for i, p := range prev {
			r := dal.NewRecord()
			id, _ := p.Get("id")
			title, _ := p.Get("title")
			r.Set("id", id)
			r.Set("title", title)
			in[i] = r
		}
		_, err := apiKeys(ctx.Container).UpsertApiKeys(ctx.Context, in)
		return err
	})

// UpdateApiKeysWorkflow mirrors updateApiKeysWorkflow.
var UpdateApiKeysWorkflow = composer.CreateWorkflow("update-api-keys",
	func(e *composer.Execution, in SelectorInput) ([]*dal.Record, error) {
		return UpdateApiKeysStep.Run(e, in)
	})

// RevokeApiKeysStep mirrors revokeApiKeysStep (no compensation).
var RevokeApiKeysStep = composer.CreateStep("revoke-api-keys",
	func(ctx composer.StepExecutionContext, in SelectorInput) (composer.StepResponse[[]*dal.Record], error) {
		out, err := apiKeys(ctx.Container).Revoke(ctx.Context, in.Selector, in.Data)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(out), nil
	}, nil)

// RevokeApiKeysWorkflow mirrors revokeApiKeysWorkflow.
var RevokeApiKeysWorkflow = composer.CreateWorkflow("revoke-api-keys",
	func(e *composer.Execution, in SelectorInput) ([]*dal.Record, error) {
		return RevokeApiKeysStep.Run(e, in)
	})

// DeleteApiKeysStep mirrors deleteApiKeysStep (no compensation).
var DeleteApiKeysStep = composer.CreateStep("delete-api-keys",
	func(ctx composer.StepExecutionContext, ids []any) (composer.StepResponse[any], error) {
		if err := apiKeys(ctx.Container).DeleteApiKeys(ctx.Context, ids); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// DeleteApiKeysWorkflow mirrors deleteApiKeysWorkflow.
var DeleteApiKeysWorkflow = composer.CreateWorkflow("delete-api-keys",
	func(e *composer.Execution, ids []any) (any, error) {
		if _, err := DeleteApiKeysStep.Run(e, ids); err != nil {
			return nil, err
		}
		keys := dal.NewRecord()
		keys.Set("publishable_key_id", ids)
		entry := dal.NewRecord()
		entry.Set(utilsmodulessdk.API_KEY, keys)
		_, err := commonsteps.RemoveRemoteLinkStep.Run(e, []*dal.Record{entry})
		return nil, err
	})

// LinkInput mirrors LinkWorkflowInput.
type LinkInput struct {
	ID     string
	Add    []string
	Remove []string
}

// ValidateSalesChannelsExistStep mirrors validateSalesChannelsExistStep.
var ValidateSalesChannelsExistStep = composer.CreateStep("validate-sales-channels-exist",
	func(ctx composer.StepExecutionContext, ids []string) (composer.StepResponse[[]any], error) {
		in := make([]any, len(ids))
		for i, id := range ids {
			in[i] = id
		}
		f := dal.NewRecord()
		f.Set("id", in)
		channels, err := commonsteps.ModuleOf(ctx.Container, utilsmodulessdk.SALES_CHANNEL).ListBySuffix(ctx.Context, "SalesChannels", f, utilsmodulessdk.FindConfig{Select: []string{"id"}})
		if err != nil {
			return composer.StepResponse[[]any]{}, err
		}
		found := map[string]bool{}
		for _, c := range channels {
			id, _ := c.Get("id")
			found[fmt.Sprint(id)] = true
		}
		var missing []string
		for _, id := range ids {
			if !found[id] {
				missing = append(missing, id)
			}
		}
		if len(missing) > 0 {
			return composer.StepResponse[[]any]{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Sales channels with IDs %s do not exist", strings.Join(missing, ", ")))
		}
		return composer.NewStepResponse(commonsteps.RecordIDs(channels)), nil
	}, nil)

type linkCompensation struct{ created, dismissed []*dal.Record }

// LinkSalesChannelsToApiKeyStep mirrors linkSalesChannelsToApiKeyStep.
var LinkSalesChannelsToApiKeyStep = composer.CreateStep("link-sales-channels-to-api-key",
	func(ctx composer.StepExecutionContext, in LinkInput) (composer.StepResponse[any], error) {
		if len(in.Add) == 0 && len(in.Remove) == 0 {
			return composer.NewStepResponse[any](nil), nil
		}
		def := func(sc string) *dal.Record {
			return modulessdk.NewLinkDefinition("api_key", "publishable_key_id", in.ID, "sales_channel", "sales_channel_id", sc)
		}
		var create, dismiss []*dal.Record
		for _, sc := range in.Add {
			create = append(create, def(sc))
		}
		for _, sc := range in.Remove {
			dismiss = append(dismiss, def(sc))
		}
		link := commonsteps.LinkOf(ctx.Container)
		if len(create) > 0 {
			if _, err := link.Create(ctx.Context, create); err != nil {
				return composer.StepResponse[any]{}, err
			}
		}
		if len(dismiss) > 0 {
			if _, err := link.Dismiss(ctx.Context, dismiss); err != nil {
				return composer.StepResponse[any]{}, err
			}
		}
		return composer.NewStepResponse[any](nil, any(linkCompensation{created: create, dismissed: dismiss})), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		c, ok := data.(linkCompensation)
		if !ok {
			return nil
		}
		link := commonsteps.LinkOf(ctx.Container)
		if len(c.created) > 0 {
			if _, err := link.Dismiss(ctx.Context, c.created); err != nil {
				return err
			}
		}
		if len(c.dismissed) > 0 {
			_, err := link.Create(ctx.Context, c.dismissed)
			return err
		}
		return nil
	})

// LinkSalesChannelsToApiKeyWorkflow mirrors linkSalesChannelsToApiKeyWorkflow.
var LinkSalesChannelsToApiKeyWorkflow = composer.CreateWorkflow("link-sales-channels-to-api-key",
	func(e *composer.Execution, in LinkInput) (any, error) {
		add := in.Add
		if add == nil {
			add = []string{}
		}
		if _, err := ValidateSalesChannelsExistStep.Run(e, add); err != nil {
			return nil, err
		}
		return LinkSalesChannelsToApiKeyStep.Run(e, in)
	})
