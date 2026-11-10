// Package steps mirrors @medusajs/core-flows/src/translation/steps.
package steps

import (
	"context"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// TranslationModule is the translation module service surface the steps
// use (ITranslationModuleService).
type TranslationModule interface {
	ListTranslations(ctx context.Context, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]*dal.Record, error)
	CreateTranslations(ctx context.Context, data []*dal.Record) ([]*dal.Record, error)
	UpdateTranslations(ctx context.Context, data []*dal.Record) ([]*dal.Record, error)
	DeleteTranslations(ctx context.Context, ids []any) error
	SoftDeleteTranslations(ctx context.Context, ids []any) error
	RestoreTranslations(ctx context.Context, ids []any) error
	ListTranslationSettings(ctx context.Context, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]*dal.Record, error)
	CreateTranslationSettings(ctx context.Context, data []*dal.Record) ([]*dal.Record, error)
	UpdateTranslationSettings(ctx context.Context, data []*dal.Record) ([]*dal.Record, error)
	DeleteTranslationSettings(ctx context.Context, ids []any) error
}

// ModuleOf resolves the translation module service.
func ModuleOf(c composer.Container) TranslationModule {
	s, _ := c.Resolve(utilsmodulessdk.TRANSLATION).(TranslationModule)
	return s
}

func ids(rs []*dal.Record) []any {
	out := make([]any, len(rs))
	for i, r := range rs {
		out[i], _ = r.Get("id")
	}
	return out
}

func idFilter(list []any) *dal.Record {
	f := dal.NewRecord()
	f.Set("id", list)
	return f
}

// CreateTranslationsStep mirrors createTranslationsStep.
var CreateTranslationsStep = composer.CreateStep("create-translations",
	func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		created, err := ModuleOf(ctx.Container).CreateTranslations(ctx.Context, data)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		if created == nil {
			created = []*dal.Record{}
		}
		return composer.NewStepResponse(created, any(ids(created))), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		created, _ := data.([]any)
		if len(created) == 0 {
			return nil
		}
		return ModuleOf(ctx.Container).DeleteTranslations(ctx.Context, created)
	})

// UpdateTranslationsStep mirrors updateTranslationsStep for the
// { translations } input.
var UpdateTranslationsStep = composer.CreateStep("update-translations",
	func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		for _, t := range data {
			if id, _ := t.Get("id"); id == nil || id == "" {
				return composer.StepResponse[[]*dal.Record]{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA,
					"Translation ID is required when doing a batch update of translations")
			}
		}
		if len(data) == 0 {
			return composer.NewStepResponse([]*dal.Record{}, any([]*dal.Record{})), nil
		}
		svc := ModuleOf(ctx.Container)
		prev, err := svc.ListTranslations(ctx.Context, idFilter(ids(data)), utilsmodulessdk.FindConfig{})
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		updated, err := svc.UpdateTranslations(ctx.Context, data)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		if updated == nil {
			updated = []*dal.Record{}
		}
		return composer.NewStepResponse(updated, any(prev)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		prev, _ := data.([]*dal.Record)
		if len(prev) == 0 {
			return nil
		}
		restore := make([]*dal.Record, len(prev))
		for i, t := range prev {
			r := dal.NewRecord()
			for _, k := range []string{"id", "reference_id", "reference", "locale_code", "translations"} {
				v, _ := t.Get(k)
				r.Set(k, v)
			}
			restore[i] = r
		}
		_, err := ModuleOf(ctx.Container).UpdateTranslations(ctx.Context, restore)
		return err
	})

// DeleteTranslationsStep mirrors deleteTranslationsStep (soft delete).
var DeleteTranslationsStep = composer.CreateStep("delete-translations",
	func(ctx composer.StepExecutionContext, list []any) (composer.StepResponse[any], error) {
		if err := ModuleOf(ctx.Container).SoftDeleteTranslations(ctx.Context, list); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil, any(list)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		list, _ := data.([]any)
		if len(list) == 0 {
			return nil
		}
		return ModuleOf(ctx.Container).RestoreTranslations(ctx.Context, list)
	})

// CreateTranslationSettingsStep mirrors createTranslationSettingsStep.
var CreateTranslationSettingsStep = composer.CreateStep("create-translation-settings",
	func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		created, err := ModuleOf(ctx.Container).CreateTranslationSettings(ctx.Context, data)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		if created == nil {
			created = []*dal.Record{}
		}
		return composer.NewStepResponse(created, any(ids(created))), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		created, _ := data.([]any)
		if len(created) == 0 {
			return nil
		}
		return ModuleOf(ctx.Container).DeleteTranslationSettings(ctx.Context, created)
	})

// UpdateTranslationSettingsStep mirrors updateTranslationSettingsStep.
var UpdateTranslationSettingsStep = composer.CreateStep("update-translation-settings",
	func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		svc := ModuleOf(ctx.Container)
		prev, err := svc.ListTranslationSettings(ctx.Context, idFilter(ids(data)), utilsmodulessdk.FindConfig{})
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		updated, err := svc.UpdateTranslationSettings(ctx.Context, data)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		if updated == nil {
			updated = []*dal.Record{}
		}
		return composer.NewStepResponse(updated, any(prev)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		prev, _ := data.([]*dal.Record)
		if len(prev) == 0 {
			return nil
		}
		_, err := ModuleOf(ctx.Container).UpdateTranslationSettings(ctx.Context, prev)
		return err
	})

// DeleteTranslationSettingsStep mirrors deleteTranslationSettingsStep.
var DeleteTranslationSettingsStep = composer.CreateStep("delete-translation-settings",
	func(ctx composer.StepExecutionContext, list []any) (composer.StepResponse[any], error) {
		svc := ModuleOf(ctx.Container)
		prev, err := svc.ListTranslationSettings(ctx.Context, idFilter(list), utilsmodulessdk.FindConfig{})
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		if err := svc.DeleteTranslationSettings(ctx.Context, list); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil, any(prev)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		prev, _ := data.([]*dal.Record)
		if len(prev) == 0 {
			return nil
		}
		_, err := ModuleOf(ctx.Container).CreateTranslationSettings(ctx.Context, prev)
		return err
	})
