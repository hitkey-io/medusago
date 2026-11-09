// Package steps mirrors @medusajs/core-flows/src/common/steps.
package steps

import (
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// UseQueryGraphStepInput mirrors UseQueryGraphStepInput.
type UseQueryGraphStepInput struct {
	query.GraphInput
	Options joiner.Options
	// IsList defaults to true; false returns data[0] (nil when empty).
	IsList *bool
}

// QueryGraphResult is the step output: data is a list, or one row when
// IsList is false.
type QueryGraphResult struct {
	Data     any
	Metadata *dal.Record
}

// QueryOf resolves Query from a container.
func QueryOf(c composer.Container) *query.Query {
	q, _ := c.Resolve(common.ContainerRegistrationKeys.QUERY).(*query.Query)
	return q
}

// UseQueryGraphStep mirrors useQueryGraphStep.
var UseQueryGraphStep = composer.CreateStep("use-query-graph-step",
	func(ctx composer.StepExecutionContext, in UseQueryGraphStepInput) (composer.StepResponse[QueryGraphResult], error) {
		res, err := QueryOf(ctx.Container).Graph(ctx.Context, in.GraphInput, in.Options)
		if err != nil {
			return composer.StepResponse[QueryGraphResult]{}, err
		}
		out := QueryGraphResult{Data: res.Data, Metadata: res.Metadata}
		if in.IsList != nil && !*in.IsList {
			if len(res.Data) > 0 {
				out.Data = res.Data[0]
			} else {
				out.Data = nil
			}
		}
		return composer.NewStepResponse(out), nil
	}, nil)

// RemoteStepInput mirrors EntryStepInput.
type RemoteStepInput struct {
	EntryPoint              string
	Fields                  []string
	Variables               *dal.Record
	ThrowIfKeyNotFound      bool
	ThrowIfRelationNotFound bool
	// List defaults to true.
	List *bool
}

// UseRemoteQueryStep mirrors useRemoteQueryStep: the rows (or the first
// row when List is false); paginated queries return { rows, metadata }.
var UseRemoteQueryStep = composer.CreateStep("use-remote-query",
	func(ctx composer.StepExecutionContext, in RemoteStepInput) (composer.StepResponse[*query.GraphResult], error) {
		obj := query.RemoteQueryObjectFromString(query.RemoteQueryObjectConfig{EntryPoint: in.EntryPoint, Fields: in.Fields, Variables: in.Variables})
		opts := joiner.Options{ThrowIfKeyNotFound: in.ThrowIfKeyNotFound}
		if in.ThrowIfKeyNotFound {
			opts.ThrowIfRelationNotFound = in.ThrowIfRelationNotFound
		}
		res, err := QueryOf(ctx.Container).RemoteQuery(ctx.Context, obj, opts)
		if err != nil {
			return composer.StepResponse[*query.GraphResult]{}, err
		}
		if in.List != nil && !*in.List {
			if len(res.Data) > 0 {
				res.Data = res.Data[:1]
			}
		}
		return composer.NewStepResponse(res), nil
	}, nil)

// RemoteQuery mirrors remoteQuery(remoteQueryObjectFromString({ entryPoint,
// variables, fields })) inside a step: the rows.
func RemoteQuery(ctx composer.StepExecutionContext, entryPoint string, variables *dal.Record, fields []string) ([]any, error) {
	obj := query.RemoteQueryObjectFromString(query.RemoteQueryObjectConfig{EntryPoint: entryPoint, Fields: fields, Variables: variables})
	res, err := QueryOf(ctx.Container).RemoteQuery(ctx.Context, obj)
	if err != nil {
		return nil, err
	}
	return res.Data, nil
}

// RemoteQueryService mirrors remoteQuery({ service: LINKS.X, variables,
// fields }) through the link's entry point.
func RemoteQueryService(ctx composer.StepExecutionContext, entryPoint string, variables *dal.Record, fields []string) ([]any, error) {
	return RemoteQuery(ctx, entryPoint, variables, fields)
}
