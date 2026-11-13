// Package workflows mirrors @medusajs/core-flows/src/promotion/workflows.
package workflows

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/promotion/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

func hook(e *composer.Execution, name string, kv ...any) error {
	r := dal.NewRecord()
	for i := 0; i+1 < len(kv); i += 2 {
		r.Set(kv[i].(string), kv[i+1])
	}
	return composer.CreateHook(e, name, r)
}

// CampaignsInput mirrors { campaignsData, additional_data }.
type CampaignsInput struct {
	CampaignsData  []*dal.Record `json:"campaignsData"`
	AdditionalData *dal.Record   `json:"additional_data,omitempty"`
}

// CreateCampaignsWorkflow mirrors createCampaignsWorkflow.
var CreateCampaignsWorkflow = composer.CreateWorkflow("create-campaigns",
	func(e *composer.Execution, in CampaignsInput) ([]*dal.Record, error) {
		created, err := steps.CreateCampaignsStep.Run(e, in.CampaignsData)
		if err != nil {
			return nil, err
		}
		return created, hook(e, "campaignsCreated", "campaigns", created, "additional_data", in.AdditionalData)
	})

// UpdateCampaignsWorkflow mirrors updateCampaignsWorkflow.
var UpdateCampaignsWorkflow = composer.CreateWorkflow("update-campaigns",
	func(e *composer.Execution, in CampaignsInput) ([]*dal.Record, error) {
		updated, err := steps.UpdateCampaignsStep.Run(e, in.CampaignsData)
		if err != nil {
			return nil, err
		}
		return updated, hook(e, "campaignsUpdated", "campaigns", updated, "additional_data", in.AdditionalData)
	})

// DeleteCampaignsWorkflow mirrors deleteCampaignsWorkflow.
var DeleteCampaignsWorkflow = composer.CreateWorkflow("delete-campaigns",
	func(e *composer.Execution, ids []any) (any, error) {
		if _, err := steps.DeleteCampaignsStep.Run(e, ids); err != nil {
			return nil, err
		}
		return nil, hook(e, "campaignsDeleted", "ids", ids)
	})

// AddOrRemoveCampaignPromotionsWorkflow mirrors
// addOrRemoveCampaignPromotionsWorkflow.
var AddOrRemoveCampaignPromotionsWorkflow = composer.CreateWorkflow("add-or-remove-campaign-promotions",
	func(e *composer.Execution, in steps.LinkInput) (any, error) {
		if _, err := steps.AddCampaignPromotionsStep.Run(e, in); err != nil {
			return nil, err
		}
		_, err := steps.RemoveCampaignPromotionsStep.Run(e, in)
		return nil, err
	})

// The Go bodies take one property of the upstream input object.
func init() {
	DeleteCampaignsWorkflow.InputFrom("ids")
}
