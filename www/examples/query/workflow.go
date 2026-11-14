package queries

// region: workflow-imports
import (
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// endregion: workflow-imports

// region: workflow
// PostsInput is the input of PostsWorkflow.
type PostsInput struct {
	Status string
}

// PostsWorkflow retrieves posts with the useQueryGraph step.
var PostsWorkflow = composer.CreateWorkflow("posts-workflow",
	func(e *composer.Execution, input PostsInput) (any, error) {
		result, err := commonsteps.UseQueryGraphStep.Run(e, commonsteps.UseQueryGraphStepInput{
			GraphInput: query.GraphInput{
				Entity:  "post",
				Fields:  []string{"id", "title", "products.*"},
				Filters: rec("status", input.Status),
			},
		})
		if err != nil {
			return nil, err
		}
		return result.Data, nil
	})

// endregion: workflow
