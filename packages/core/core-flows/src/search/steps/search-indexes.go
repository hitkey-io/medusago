// Package steps mirrors @medusajs/core-flows/src/search/steps.
package steps

import (
	"context"
	"fmt"

	searchtypes "github.com/hitkey-io/medusago/packages/core/types/src/search"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// SearchIndexesInput mirrors the index selection of the search steps: one
// name, several, or none (every registered index).
type SearchIndexesInput struct {
	Index []string
	// Since, Filters and Strategy mirror the reindex options.
	Since    *string
	Filters  *dal.Record
	Strategy *string
}

func resolveSearch(c composer.Container) any {
	return c.Resolve(utilsmodulessdk.SEARCH)
}

// indexResolver is the Search Module's getIndex (throws NOT_FOUND).
type indexResolver interface {
	ResolveIndex(name string) error
}

// ValidateSearchIndexesExistStep mirrors validateSearchIndexesExistStep:
// every requested index must be registered (getIndex throws NOT_FOUND).
var ValidateSearchIndexesExistStep = composer.CreateStep("validate-search-indexes-exist",
	func(sc composer.StepExecutionContext, in SearchIndexesInput) (composer.StepResponse[any], error) {
		if len(in.Index) == 0 {
			return composer.NewStepResponse[any](nil), nil
		}
		svc, _ := resolveSearch(sc.Container).(indexResolver)
		for _, name := range in.Index {
			if svc == nil {
				return composer.StepResponse[any]{}, fmt.Errorf("Cannot read properties of undefined (reading 'getIndex')")
			}
			if err := svc.ResolveIndex(name); err != nil {
				return composer.StepResponse[any]{}, err
			}
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

type indexDeleter interface {
	DeleteIndex(ctx context.Context, name string) (*dal.Record, error)
}

// DeleteSearchIndexStep mirrors deleteSearchIndexStep (no compensation).
var DeleteSearchIndexStep = composer.CreateStep("delete-search-index",
	func(sc composer.StepExecutionContext, in SearchIndexesInput) (composer.StepResponse[*dal.Record], error) {
		svc, _ := resolveSearch(sc.Container).(indexDeleter)
		name := ""
		if len(in.Index) > 0 {
			name = in.Index[0]
		}
		result, err := svc.DeleteIndex(sc.Context, name)
		if err != nil {
			return composer.StepResponse[*dal.Record]{}, err
		}
		return composer.NewStepResponse(result), nil
	}, nil)

type indexReindexer interface {
	Reindex(ctx context.Context, input searchtypes.ReindexInput) (searchtypes.ReindexResult, error)
}

// ReindexSearchIndexesStep mirrors reindexSearchIndexesStep: an
// unparseable `since` fails here rather than rebuilding nothing.
var ReindexSearchIndexesStep = composer.CreateStep("reindex-search-indexes",
	func(sc composer.StepExecutionContext, in SearchIndexesInput) (composer.StepResponse[*dal.Record], error) {
		svc, _ := resolveSearch(sc.Container).(indexReindexer)
		input := searchtypes.ReindexInput{Index: in.Index, Filters: in.Filters}
		if in.Since != nil && *in.Since != "" {
			since, ok := common.ParseJSDate(*in.Since)
			if !ok {
				return composer.StepResponse[*dal.Record]{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf(`"%s" is not a valid date to reindex since`, *in.Since))
			}
			input.Since = &since
		}
		if in.Strategy != nil {
			input.Strategy = *in.Strategy
		}
		if svc == nil {
			return composer.StepResponse[*dal.Record]{}, fmt.Errorf("Cannot read properties of undefined (reading 'reindex')")
		}
		result, err := svc.Reindex(sc.Context, input)
		if err != nil {
			return composer.StepResponse[*dal.Record]{}, err
		}
		out := dal.NewRecord()
		out.Set("job_id", result.JobID)
		indexes := make([]any, len(result.Indexes))
		for i, name := range result.Indexes {
			indexes[i] = name
		}
		out.Set("indexes", indexes)
		return composer.NewStepResponse(out), nil
	}, nil)
