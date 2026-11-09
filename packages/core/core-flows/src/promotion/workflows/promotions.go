package workflows

import (
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/promotion/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// PromotionsInput mirrors { promotionsData, additional_data }.
type PromotionsInput struct {
	PromotionsData []*dal.Record
	AdditionalData *dal.Record
}

// CreatePromotionsWorkflow mirrors createPromotionsWorkflow.
var CreatePromotionsWorkflow = composer.CreateWorkflow("create-promotions",
	func(e *composer.Execution, in PromotionsInput) ([]*dal.Record, error) {
		created, err := steps.CreatePromotionsStep.Run(e, in.PromotionsData)
		if err != nil {
			return nil, err
		}
		return created, hook(e, "promotionsCreated", "promotions", created, "additional_data", in.AdditionalData)
	})

// UpdatePromotionsStatusWorkflow mirrors updatePromotionsStatusWorkflow.
var UpdatePromotionsStatusWorkflow = composer.CreateWorkflow("update-promotions-status",
	func(e *composer.Execution, in PromotionsInput) ([]*dal.Record, error) {
		if _, err := steps.UpdatePromotionsValidationStep.Run(e, in.PromotionsData); err != nil {
			return nil, err
		}
		updated, err := steps.UpdatePromotionsStep.Run(e, in.PromotionsData)
		if err != nil {
			return nil, err
		}
		return updated, hook(e, "promotionStatusUpdated", "promotions", updated, "additional_data", in.AdditionalData)
	})

// UpdatePromotionsWorkflow mirrors updatePromotionsWorkflow.
var UpdatePromotionsWorkflow = composer.CreateWorkflow("update-promotions",
	func(e *composer.Execution, in PromotionsInput) ([]*dal.Record, error) {
		ids := make([]any, len(in.PromotionsData))
		for i, d := range in.PromotionsData {
			ids[i], _ = d.Get("id")
		}
		vars := dal.NewRecord()
		vars.Set("id", ids)
		res, err := commonsteps.UseRemoteQueryStep.Config("get-promotions", composer.StepOptions{}).Run(e, commonsteps.RemoteStepInput{
			EntryPoint: "promotion", Variables: vars, Fields: []string{"id", "status"}, ThrowIfKeyNotFound: true,
		})
		if err != nil {
			return nil, err
		}
		statusByID := map[any]any{}
		for _, row := range res.Data {
			if r, ok := row.(*dal.Record); ok {
				id, _ := r.Get("id")
				status, _ := r.Get("status")
				statusByID[id] = status
			}
		}
		var updates, statusUpdates []*dal.Record
		for _, d := range in.PromotionsData {
			rest := dal.NewRecord()
			for _, k := range d.Keys() {
				if k == "status" {
					continue
				}
				v, _ := d.Get(k)
				rest.Set(k, v)
			}
			updates = append(updates, rest)
			id, _ := d.Get("id")
			if status, ok := d.Get("status"); ok {
				if s, isStr := status.(string); isStr && s != statusByID[id] {
					su := dal.NewRecord()
					su.Set("id", id)
					su.Set("status", s)
					statusUpdates = append(statusUpdates, su)
				}
			}
		}
		updated, err := steps.UpdatePromotionsStep.Run(e, updates)
		if err != nil {
			return nil, err
		}
		if len(statusUpdates) > 0 {
			if _, err := UpdatePromotionsStatusWorkflow.RunAsStep(e, PromotionsInput{PromotionsData: statusUpdates}); err != nil {
				return nil, err
			}
		}
		return updated, hook(e, "promotionsUpdated", "promotions", updated, "additional_data", in.AdditionalData)
	})

// DeletePromotionsWorkflow mirrors deletePromotionsWorkflow.
var DeletePromotionsWorkflow = composer.CreateWorkflow("delete-promotions",
	func(e *composer.Execution, ids []any) (any, error) {
		if _, err := steps.DeletePromotionsStep.Run(e, ids); err != nil {
			return nil, err
		}
		return nil, hook(e, "promotionsDeleted", "ids", ids)
	})

// CreatePromotionRulesWorkflow mirrors createPromotionRulesWorkflow.
var CreatePromotionRulesWorkflow = composer.CreateWorkflow("create-promotion-rules-workflow",
	func(e *composer.Execution, in steps.RulesInput) ([]*dal.Record, error) {
		return steps.AddRulesToPromotionsStep.Run(e, in)
	})

// UpdatePromotionRulesWorkflow mirrors updatePromotionRulesWorkflow.
var UpdatePromotionRulesWorkflow = composer.CreateWorkflow("update-promotion-rules-workflow",
	func(e *composer.Execution, data []*dal.Record) ([]*dal.Record, error) {
		return steps.UpdatePromotionRulesStep.Run(e, data)
	})

// DeletePromotionRulesWorkflow mirrors deletePromotionRulesWorkflow.
var DeletePromotionRulesWorkflow = composer.CreateWorkflow("delete-promotion-rules-workflow",
	func(e *composer.Execution, in steps.RulesInput) (any, error) {
		_, err := steps.RemoveRulesFromPromotionsStep.Run(e, in)
		return nil, err
	})

// BatchPromotionRulesInput mirrors BatchPromotionRulesWorkflowInput.
type BatchPromotionRulesInput struct {
	ID       any
	RuleType steps.RuleType
	Create   []*dal.Record
	Update   []*dal.Record
	Delete   []any
}

// BatchPromotionRulesResult mirrors BatchPromotionRulesWorkflowOutput.
type BatchPromotionRulesResult struct {
	Created []*dal.Record
	Updated []*dal.Record
	Deleted []any
}

// BatchPromotionRulesWorkflow mirrors batchPromotionRulesWorkflow (the
// delete runs deletePromotionRulesWorkflow as a nested step).
var BatchPromotionRulesWorkflow = composer.CreateWorkflow("batch-promotion-rules",
	func(e *composer.Execution, in BatchPromotionRulesInput) (BatchPromotionRulesResult, error) {
		create := in.Create
		if create == nil {
			create = []*dal.Record{}
		}
		update := in.Update
		if update == nil {
			update = []*dal.Record{}
		}
		del := in.Delete
		if del == nil {
			del = []any{}
		}
		var created, updated []*dal.Record
		err := composer.Parallelize(
			func() (err error) {
				created, err = CreatePromotionRulesWorkflow.RunAsStep(e, steps.RulesInput{RuleType: in.RuleType, ID: in.ID, Rules: create})
				return err
			},
			func() (err error) {
				updated, err = UpdatePromotionRulesWorkflow.RunAsStep(e, update)
				return err
			},
			func() error {
				_, err := DeletePromotionRulesWorkflow.RunAsStep(e, steps.RulesInput{RuleType: in.RuleType, ID: in.ID, RuleIDs: del})
				return err
			},
		)
		if err != nil {
			return BatchPromotionRulesResult{}, err
		}
		return BatchPromotionRulesResult{Created: created, Updated: updated, Deleted: del}, nil
	})
