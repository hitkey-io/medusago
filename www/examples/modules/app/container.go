package main

// region: container-imports
import (
	"context"
	"log/slog"

	cli "github.com/hitkey-io/medusago/packages/cli/medusa-cli/src"
	"github.com/hitkey-io/medusago/packages/core/framework/src/container"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	"github.com/hitkey-io/medusago/packages/medusa/src/loaders"
	"github.com/hitkey-io/medusago/www/examples/modules/blog"
)

// endregion: container-imports

// region: container-route
// listPostsRoute resolves resources from the request scope.
func listPostsRoute(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	logger := req.Scope.Resolve(common.ContainerRegistrationKeys.LOGGER).(*slog.Logger)
	blogService := req.Scope.Resolve(blog.Name).(*blog.BlogModuleService)

	logger.Info("Listing posts")
	posts, err := blogService.Internal("Post").List(req.Ctx(), dal.NewRecord(), utilsmodulessdk.FindConfig{})
	if err != nil {
		return err
	}

	body := dal.NewRecord()
	body.Set("posts", posts)
	return res.JSON(body)
}

// endregion: container-route

// region: container-step
// listPostsStep resolves resources from the container of the workflow.
var listPostsStep = composer.CreateStep("list-posts-step",
	func(ctx composer.StepExecutionContext, _ any) (composer.StepResponse[[]*dal.Record], error) {
		blogService := ctx.Container.Resolve(blog.Name).(*blog.BlogModuleService)

		posts, err := blogService.Internal("Post").List(ctx.Context, dal.NewRecord(), utilsmodulessdk.FindConfig{})
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(posts), nil
	}, nil)

// endregion: container-step

// region: container-has
func hasEventBus(c *container.MedusaContainer) bool {
	// Has reports whether something is registered under a key. Resolve
	// returns nil for a key that is not registered.
	return c.Has(utilsmodulessdk.EVENT_BUS)
}

// endregion: container-has

// region: container-register
// Greeter is a resource of the project.
type Greeter struct{ Greeting string }

// registerResources registers the resources of the project in the container
// once the application is loaded. Pass it in the OnLoad option of cli.Run.
func registerResources(_ context.Context, app *loaders.MedusaApp) error {
	app.Container.Register("greeter", &Greeter{Greeting: "Hello"})
	return nil
}

// endregion: container-register

// region: container-register-use
// greet resolves the resource that registerResources registered.
func greet(c *container.MedusaContainer, name string) string {
	greeter := c.Resolve("greeter").(*Greeter)
	return greeter.Greeting + ", " + name
}

// endregion: container-register-use

// region: container-onload
// runWithResources runs the CLI with the hook that registers the resources.
func runWithResources(args []string) int {
	return cli.Run(args, cli.Options{
		Config: Config,
		OnLoad: []func(ctx context.Context, app *loaders.MedusaApp) error{registerResources},
	})
}

// endregion: container-onload
