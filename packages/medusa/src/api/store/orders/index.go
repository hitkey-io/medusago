package orders

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/framework/src/http/middlewares"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

// StoreGetOrderParams mirrors StoreGetOrderParams.
var StoreGetOrderParams = utils.CreateSelectParams()

// StoreGetOrdersParamsFields mirrors StoreGetOrdersParamsFields.
var StoreGetOrdersParamsFields = zod.Object(
	zod.F("id", zod.Union(zod.String(), zod.Array(zod.String())).Optional()),
	zod.F("status", zod.Union(zod.String(), zod.Array(zod.String())).Optional()),
)

// StoreGetOrdersParams mirrors StoreGetOrdersParams.
var StoreGetOrdersParams = utils.CreateFindParams(utils.FindParamsOptions{Offset: utils.Num(0), Limit: utils.Num(50)}).
	Merge(StoreGetOrdersParamsFields).
	Merge(commonvalidators.ApplyAndAndOrOperators(StoreGetOrdersParamsFields))

func mw(fns ...medusahttp.MiddlewareFunction) []medusahttp.MiddlewareFunction { return fns }

// Middlewares mirrors storeOrderRoutesMiddlewares (ported routes only).
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/store/orders", Middlewares: mw(
		middlewares.Authenticate([]string{"customer"}, []string{"session", "bearer"}, medusahttp.AuthenticateOptions{}),
		httputils.ValidateAndTransformQuery(StoreGetOrdersParams, ListConfig),
	)},
	{Methods: []string{"GET"}, Matcher: "/store/orders/:id", Middlewares: mw(httputils.ValidateAndTransformQuery(StoreGetOrderParams, RetrieveConfig))},
	{Methods: []string{"POST"}, Matcher: "/store/orders/:id/transfer/request", Middlewares: mw(
		middlewares.Authenticate([]string{"customer"}, []string{"session", "bearer"}, medusahttp.AuthenticateOptions{}),
		httputils.ValidateAndTransformBody(httputils.Schema(StoreRequestOrderTransfer)),
		httputils.ValidateAndTransformQuery(StoreGetOrderParams, RetrieveConfig),
	)},
	{Methods: []string{"POST"}, Matcher: "/store/orders/:id/transfer/cancel", Middlewares: mw(
		middlewares.Authenticate([]string{"customer"}, []string{"session", "bearer"}, medusahttp.AuthenticateOptions{}),
		httputils.ValidateAndTransformQuery(StoreGetOrderParams, RetrieveConfig),
	)},
	{Methods: []string{"POST"}, Matcher: "/store/orders/:id/transfer/accept", Middlewares: mw(
		httputils.ValidateAndTransformBody(httputils.Schema(StoreAcceptOrderTransfer)),
		httputils.ValidateAndTransformQuery(StoreGetOrderParams, RetrieveConfig),
	)},
	{Methods: []string{"POST"}, Matcher: "/store/orders/:id/transfer/decline", Middlewares: mw(
		httputils.ValidateAndTransformBody(httputils.Schema(StoreDeclineOrderTransferRequest)),
		httputils.ValidateAndTransformQuery(StoreGetOrderParams, RetrieveConfig),
	)},
}

// StoreAcceptOrderTransfer mirrors StoreAcceptOrderTransfer.
var StoreAcceptOrderTransfer = zod.Object(zod.F("token", zod.String().Min(1)))

// StoreRequestOrderTransfer mirrors StoreRequestOrderTransfer.
var StoreRequestOrderTransfer = zod.Object(
	zod.F("description", zod.String().Optional()),
	zod.F("update_order_email", zod.Boolean().Optional()),
)

// StoreDeclineOrderTransferRequest mirrors StoreDeclineOrderTransferRequest.
var StoreDeclineOrderTransferRequest = zod.Object(zod.F("token", zod.String().Min(1)))

// GET mirrors GET /store/orders.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	filters := dal.NewRecord()
	if req.FilterableFields != nil {
		for _, k := range req.FilterableFields.Keys() {
			v, _ := req.FilterableFields.Get(k)
			filters.Set(k, v)
		}
	}
	filters.Set("is_draft_order", false)
	var actorID any
	if req.AuthContext != nil {
		actorID, _ = req.AuthContext.Get("actor_id")
	}
	filters.Set("customer_id", actorID)
	r, err := workflows.GetOrdersList(req.Ctx(), httputils.QueryOf(req.Scope), workflows.GetOrdersListInput{
		Fields:    req.QueryConfig.Fields,
		Variables: httputils.ListVariables(filters, req.QueryConfig.Pagination),
	})
	if err != nil {
		return err
	}
	if r.Metadata == nil {
		r.Metadata = dal.NewRecord()
	}
	return res.JSON(httputils.ListResponse("orders", r))
}

// Route is route.ts.
var Route = medusahttp.Route{GET: GET}
