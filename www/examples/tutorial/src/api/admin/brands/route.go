package brands

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/www/examples/tutorial/src/workflows"
)

// region: post
// POST creates a brand with createBrandWorkflow.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	// the body was validated by the validation middleware
	body := req.ValidatedBody.(*dal.Record)
	name, _ := body.Get("name")

	brand, err := workflows.CreateBrandWorkflow.Run(req.Ctx(), req.Scope,
		workflows.CreateBrandWorkflowInput{Name: name.(string)})
	if err != nil {
		return err
	}

	out := dal.NewRecord()
	out.Set("brand", brand)
	return res.JSON(out)
}

// endregion: post

// region: get
// GET lists the brands with their linked products.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	q := httputils.QueryOf(req.Scope)

	result, err := q.Graph(req.Ctx(), query.GraphInput{
		Entity: "brand",
		Fields: []string{"*", "products.*"},
	}, joiner.Options{})
	if err != nil {
		return err
	}

	out := dal.NewRecord()
	out.Set("brands", result.Data)
	return res.JSON(out)
}

// endregion: get

// region: route
// Route holds the handlers of the route file. Its path is set where the
// route is registered, in src/api/routes.go.
var Route = medusahttp.Route{GET: GET, POST: POST}

// endregion: route
