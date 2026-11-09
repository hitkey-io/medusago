// Package steps mirrors @medusajs/core-flows/src/promotion/steps.
package steps

import (
	"context"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// PromotionService is the part of the promotion module the steps use.
type PromotionService interface {
	CreateCampaigns(ctx context.Context, data []*dal.Record) ([]*dal.Record, error)
	UpdateCampaigns(ctx context.Context, data []*dal.Record) ([]*dal.Record, error)
	AddPromotionsToCampaign(ctx context.Context, campaignID any, promotionIDs []any) ([]any, error)
	RemovePromotionsFromCampaign(ctx context.Context, campaignID any, promotionIDs []any) ([]any, error)
	CreatePromotions(ctx context.Context, data []*dal.Record) ([]*dal.Record, error)
	UpdatePromotions(ctx context.Context, data []*dal.Record) ([]*dal.Record, error)
	UpdatePromotionRules(ctx context.Context, data []*dal.Record) ([]*dal.Record, error)
	AddPromotionRules(ctx context.Context, promotionID any, rules []*dal.Record) ([]*dal.Record, error)
	AddPromotionTargetRules(ctx context.Context, promotionID any, rules []*dal.Record) ([]*dal.Record, error)
	AddPromotionBuyRules(ctx context.Context, promotionID any, rules []*dal.Record) ([]*dal.Record, error)
	RemovePromotionRules(ctx context.Context, promotionID any, ruleIDs []any) error
	RemovePromotionTargetRules(ctx context.Context, promotionID any, ruleIDs []any) error
	RemovePromotionBuyRules(ctx context.Context, promotionID any, ruleIDs []any) error
	ListBySuffix(ctx context.Context, suffix string, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]*dal.Record, error)
	Internal(className string) *utilsmodulessdk.InternalService
}

// Promotion resolves the promotion module.
func Promotion(c composer.Container) PromotionService {
	return c.Resolve(utilsmodulessdk.PROMOTION).(PromotionService)
}

func get(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}

// CreateCampaignsStep mirrors createCampaignsStep.
var CreateCampaignsStep = composer.CreateStep("create-campaigns",
	func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		created, err := Promotion(ctx.Container).CreateCampaigns(ctx.Context, data)
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
		_, err := Promotion(ctx.Container).Internal("Campaign").Delete(ctx.Context, ids)
		return err
	})

type campaignsBefore struct {
	before []*dal.Record
	keys   map[string]bool
}

// UpdateCampaignsStep mirrors updateCampaignsStep.
var UpdateCampaignsStep = composer.CreateStep("update-campaigns",
	func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		if len(data) == 0 {
			return composer.NewStepResponse([]*dal.Record{}, any(campaignsBefore{})), nil
		}
		svc := Promotion(ctx.Container)
		ids := make([]any, len(data))
		keys := map[string]bool{"id": true}
		for i, d := range data {
			ids[i] = get(d, "id")
			for _, k := range d.Keys() {
				keys[k] = true
			}
		}
		f := dal.NewRecord()
		f.Set("id", ids)
		var relations []string
		if keys["budget"] {
			relations = []string{"budget"}
		}
		before, err := svc.ListBySuffix(ctx.Context, "Campaigns", f, utilsmodulessdk.FindConfig{Relations: relations})
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		updated, err := svc.UpdateCampaigns(ctx.Context, data)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(updated, any(campaignsBefore{before: before, keys: keys})), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		c, ok := data.(campaignsBefore)
		if !ok || len(c.before) == 0 {
			return nil
		}
		restore := make([]*dal.Record, len(c.before))
		for i, b := range c.before {
			r := dal.NewRecord()
			for _, k := range b.Keys() {
				if !c.keys[k] {
					continue
				}
				v, _ := b.Get(k)
				if k == "budget" {
					br, _ := v.(*dal.Record)
					if br == nil {
						continue
					}
					nb := dal.NewRecord()
					nb.Set("limit", get(br, "limit"))
					v = nb
				}
				r.Set(k, v)
			}
			restore[i] = r
		}
		_, err := Promotion(ctx.Container).UpdateCampaigns(ctx.Context, restore)
		return err
	})

// DeleteCampaignsStep mirrors deleteCampaignsStep.
var DeleteCampaignsStep = commonsteps.SoftDeleteRecordsStep("delete-campaigns", utilsmodulessdk.PROMOTION, "Campaign")

// LinkInput mirrors LinkWorkflowInput.
type LinkInput struct {
	ID     any
	Add    []any
	Remove []any
}

// AddCampaignPromotionsStep mirrors addCampaignPromotionsStep.
var AddCampaignPromotionsStep = composer.CreateStep("add-campaign-promotions",
	func(ctx composer.StepExecutionContext, in LinkInput) (composer.StepResponse[any], error) {
		if len(in.Add) > 0 {
			if _, err := Promotion(ctx.Container).AddPromotionsToCampaign(ctx.Context, in.ID, in.Add); err != nil {
				return composer.StepResponse[any]{}, err
			}
		}
		return composer.NewStepResponse[any](nil, any(in)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		in, ok := data.(LinkInput)
		if !ok || len(in.Add) == 0 {
			return nil
		}
		_, err := Promotion(ctx.Container).RemovePromotionsFromCampaign(ctx.Context, in.ID, in.Add)
		return err
	})

// RemoveCampaignPromotionsStep mirrors removeCampaignPromotionsStep.
var RemoveCampaignPromotionsStep = composer.CreateStep("remove-campaign-promotions",
	func(ctx composer.StepExecutionContext, in LinkInput) (composer.StepResponse[any], error) {
		if len(in.Remove) > 0 {
			if _, err := Promotion(ctx.Container).RemovePromotionsFromCampaign(ctx.Context, in.ID, in.Remove); err != nil {
				return composer.StepResponse[any]{}, err
			}
		}
		return composer.NewStepResponse[any](nil, any(in)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		in, ok := data.(LinkInput)
		if !ok || len(in.Remove) == 0 {
			return nil
		}
		_, err := Promotion(ctx.Container).AddPromotionsToCampaign(ctx.Context, in.ID, in.Remove)
		return err
	})
