// Package workflows mirrors @medusajs/core-flows/src/defaults/workflows.
package workflows

import (
	apikeyworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/api-key/workflows"
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/defaults/steps"
	saleschannelsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/sales-channel/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// CreateDefaultsWorkflow mirrors createDefaultsWorkflow (run by the loaders
// at every start): the default sales channel, the default store and, when
// none exists, a publishable API key linked to the sales channel.
var CreateDefaultsWorkflow = composer.CreateWorkflow("create-defaults",
	func(e *composer.Execution, _ any) (*dal.Record, error) {
		sc := dal.NewRecord()
		sc.Set("name", "Default Sales Channel")
		sc.Set("description", "Created by Medusa")
		channel, err := saleschannelsteps.CreateDefaultSalesChannelStep.Run(e, sc)
		if err != nil {
			return nil, err
		}
		var channelID any
		if channel != nil {
			channelID, _ = channel.Get("id")
		}
		storeIn := dal.NewRecord()
		storeIn.Set("default_sales_channel_id", channelID)
		store, err := steps.CreateDefaultStoreStep.Run(e, storeIn)
		if err != nil {
			return nil, err
		}
		f := dal.NewRecord()
		f.Set("type", "publishable")
		isList := false
		res, err := commonsteps.UseQueryGraphStep.Run(e, commonsteps.UseQueryGraphStepInput{
			GraphInput: query.GraphInput{Entity: "api_key", Fields: []string{"id", "type"}, Filters: f},
			IsList:     &isList,
		})
		if err != nil {
			return nil, err
		}
		missing := res.Data == nil
		if r, ok := res.Data.(*dal.Record); ok && r == nil {
			missing = true
		}
		if missing {
			key := dal.NewRecord()
			key.Set("title", "Default Publishable API Key")
			key.Set("type", "publishable")
			key.Set("created_by", "")
			created, err := apikeyworkflows.CreateApiKeysStep.Run(e, []*dal.Record{key})
			if err != nil {
				return nil, err
			}
			keyID, _ := created[0].Get("id")
			add := []string{}
			if s, ok := channelID.(string); ok {
				add = append(add, s)
			}
			id, _ := keyID.(string)
			if _, err := apikeyworkflows.LinkSalesChannelsToApiKeyWorkflow.RunAsStep(e, apikeyworkflows.LinkInput{ID: id, Add: add}); err != nil {
				return nil, err
			}
		}
		return store, nil
	})
