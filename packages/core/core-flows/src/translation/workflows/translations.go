// Package workflows mirrors @medusajs/core-flows/src/translation/workflows.
package workflows

import (
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/translation/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// TranslationWorkflowEvents mirrors TranslationWorkflowEvents.
var TranslationWorkflowEvents = struct{ CREATED, UPDATED, DELETED string }{
	CREATED: "translation.created",
	UPDATED: "translation.updated",
	DELETED: "translation.deleted",
}

func idEvents(list []any) []any {
	out := make([]any, len(list))
	for i, id := range list {
		r := dal.NewRecord()
		r.Set("id", id)
		out[i] = r
	}
	return out
}

func recordIDEvents(rs []*dal.Record) []any {
	return idEvents(commonsteps.RecordIDs(rs))
}

// CreateTranslationsWorkflow mirrors createTranslationsWorkflow.
var CreateTranslationsWorkflow = composer.CreateWorkflow("create-translations",
	func(e *composer.Execution, translations []*dal.Record) ([]*dal.Record, error) {
		created, err := steps.CreateTranslationsStep.Run(e, translations)
		if err != nil {
			return nil, err
		}
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: TranslationWorkflowEvents.CREATED, Data: recordIDEvents(created)}); err != nil {
			return nil, err
		}
		return created, nil
	})

// UpdateTranslationsWorkflow mirrors updateTranslationsWorkflow for the
// { translations } input.
var UpdateTranslationsWorkflow = composer.CreateWorkflow("update-translations",
	func(e *composer.Execution, translations []*dal.Record) ([]*dal.Record, error) {
		updated, err := steps.UpdateTranslationsStep.Run(e, translations)
		if err != nil {
			return nil, err
		}
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: TranslationWorkflowEvents.UPDATED, Data: recordIDEvents(updated)}); err != nil {
			return nil, err
		}
		return updated, nil
	})

// DeleteTranslationsWorkflow mirrors deleteTranslationsWorkflow.
var DeleteTranslationsWorkflow = composer.CreateWorkflow("delete-translations",
	func(e *composer.Execution, ids []any) (any, error) {
		if _, err := steps.DeleteTranslationsStep.Run(e, ids); err != nil {
			return nil, err
		}
		_, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: TranslationWorkflowEvents.DELETED, Data: idEvents(ids)})
		return nil, err
	})

// BatchInput mirrors BatchTranslationsWorkflowInput and
// BatchTranslationSettingsWorkflowInput.
type BatchInput struct {
	Create []*dal.Record
	Update []*dal.Record
	Delete []any
}

// BatchResult is { created, updated, deleted }.
type BatchResult struct {
	Created []*dal.Record
	Updated []*dal.Record
}

// BatchTranslationsWorkflow mirrors batchTranslationsWorkflow.
var BatchTranslationsWorkflow = composer.CreateWorkflow("batch-translations",
	func(e *composer.Execution, in BatchInput) (BatchResult, error) {
		var out BatchResult
		err := composer.Parallelize(
			func() error {
				var err error
				out.Created, err = CreateTranslationsWorkflow.RunAsStep(e, in.Create)
				return err
			},
			func() error {
				var err error
				out.Updated, err = UpdateTranslationsWorkflow.RunAsStep(e, in.Update)
				return err
			},
			func() error {
				_, err := DeleteTranslationsWorkflow.RunAsStep(e, in.Delete)
				return err
			},
		)
		return out, err
	})

// BatchTranslationSettingsWorkflow mirrors batchTranslationSettingsWorkflow.
var BatchTranslationSettingsWorkflow = composer.CreateWorkflow("batch-translation-settings",
	func(e *composer.Execution, in BatchInput) (BatchResult, error) {
		var out BatchResult
		err := composer.Parallelize(
			func() error {
				var err error
				out.Created, err = steps.CreateTranslationSettingsStep.Run(e, in.Create)
				return err
			},
			func() error {
				var err error
				out.Updated, err = steps.UpdateTranslationSettingsStep.Run(e, in.Update)
				return err
			},
			func() error {
				_, err := steps.DeleteTranslationSettingsStep.Run(e, in.Delete)
				return err
			},
		)
		return out, err
	})
