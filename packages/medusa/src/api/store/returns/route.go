// Package returns mirrors api/store/returns.
package returns

import (
	orderworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	storeutils "github.com/hitkey-io/medusago/packages/medusa/src/api/store/utils"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

// DefaultReturnFields mirrors defaultReturnFields.
var DefaultReturnFields = []string{"id", "order_id", "created_at", "updated_at", "deleted_at", "metadata"}

// RetrieveConfig mirrors retrieveTransformQueryConfig.
var RetrieveConfig = httputils.QueryConfig{Defaults: DefaultReturnFields, Allowed: storeutils.BuildAllowedFields(DefaultReturnFields, []string{"display_id", "items", "received_at", "created_by", "canceled_at", "requested_at"})}

var returnsParamsFields = zod.Object(
	zod.F("id", zod.Union(zod.String(), zod.Array(zod.String())).Optional()),
	zod.F("order_id", zod.Union(zod.String(), zod.Array(zod.String())).Optional()),
)

// ReturnsParams mirrors ReturnsParams.
var ReturnsParams = utils.CreateFindParams(utils.FindParamsOptions{}).Merge(returnsParamsFields).Merge(commonvalidators.ApplyAndAndOrOperators(returnsParamsFields))

// StorePostReturnsReqSchema mirrors StorePostReturnsReqSchema.
var StorePostReturnsReqSchema = zod.Object(
	zod.F("order_id", zod.String().Min(1)),
	zod.F("items", zod.Array(zod.Object(
		zod.F("id", zod.String()),
		zod.F("quantity", zod.Number().Min(1)),
		zod.F("reason_id", zod.String().Nullish()),
		zod.F("note", zod.String().Nullish()),
	))),
	zod.F("return_shipping", zod.Object(
		zod.F("option_id", zod.String()),
		zod.F("price", zod.Number().Optional()),
	)),
	zod.F("note", zod.String().Nullish()),
	zod.F("receive_now", zod.Boolean().Optional()),
	zod.F("location_id", zod.String().Nullish()),
).Strict()

// Middlewares mirrors storeReturnsRoutesMiddlewares.
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"POST"}, Matcher: "/store/returns", Middlewares: []medusahttp.MiddlewareFunction{
		httputils.ValidateAndTransformBody(httputils.Schema(StorePostReturnsReqSchema)),
		httputils.ValidateAndTransformQuery(ReturnsParams, RetrieveConfig),
	}},
}

// POST mirrors POST: createAndCompleteReturnOrderWorkflow with the body.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	input, _ := req.ValidatedBody.(*dal.Record)
	result, err := orderworkflows.CreateAndCompleteReturnOrderWorkflow.Run(req.Ctx(), req.Scope, input)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("return", result)
	return res.Status(200).JSON(out)
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
