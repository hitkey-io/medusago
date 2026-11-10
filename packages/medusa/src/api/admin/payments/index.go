// Package payments mirrors api/admin/payments.
package payments

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

// DefaultAdminPaymentFields mirrors defaultAdminPaymentFields.
var DefaultAdminPaymentFields = []string{"id", "currency_code", "amount", "captured_at", "payment_collection_id", "payment_session_id", "captures.id", "captures.amount",
	"refunds.id", "refunds.amount", "refunds.note", "refunds.payment_id", "refunds.refund_reason.label", "refunds.refund_reason.code"}

// Query configs.
var (
	ListConfig          = httputils.QueryConfig{Defaults: DefaultAdminPaymentFields, IsList: true, Entity: "payment"}
	RetrieveConfig      = httputils.QueryConfig{Defaults: DefaultAdminPaymentFields, Entity: "payment"}
	ProvidersListConfig = httputils.QueryConfig{Defaults: []string{"id", "is_enabled"}, IsList: true}
)

// AdminGetPaymentParams mirrors AdminGetPaymentParams.
var AdminGetPaymentParams = utils.CreateSelectParams()

var paymentsFields = zod.Object(
	zod.F("q", zod.String().Optional()),
	zod.F("id", utils.StringOrStrings().Optional()),
	zod.F("payment_session_id", utils.StringOrStrings().Optional()),
	zod.F("created_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("updated_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("deleted_at", utils.CreateOperatorMap(nil, nil).Optional()),
)

// AdminGetPaymentsParams mirrors AdminGetPaymentsParams.
var AdminGetPaymentsParams = utils.CreateFindParams(utils.FindParamsOptions{Limit: utils.Num(20), Offset: utils.Num(0)}).Merge(paymentsFields).Merge(commonvalidators.ApplyAndAndOrOperators(paymentsFields))

var providersFields = zod.Object(
	zod.F("id", utils.StringOrStrings().Optional()),
	zod.F("is_enabled", commonvalidators.BooleanString().Optional()),
)

// AdminGetPaymentProvidersParams mirrors AdminGetPaymentProvidersParams.
var AdminGetPaymentProvidersParams = utils.CreateFindParams(utils.FindParamsOptions{Limit: utils.Num(20), Offset: utils.Num(0)}).Merge(providersFields).Merge(commonvalidators.ApplyAndAndOrOperators(providersFields))

// AdminCreatePaymentCapture mirrors AdminCreatePaymentCapture.
var AdminCreatePaymentCapture = zod.Object(zod.F("amount", zod.Number().Gt(0).Optional())).Strict()

// AdminCreatePaymentRefund mirrors AdminCreatePaymentRefund.
var AdminCreatePaymentRefund = zod.Object(
	zod.F("amount", zod.Number().Gt(0).Optional()),
	zod.F("refund_reason_id", zod.String().Optional()),
	zod.F("note", zod.String().Optional()),
	zod.F("idempotency_key", zod.String().Min(1).Max(255).Optional()),
).Strict()

func mw(fns ...medusahttp.MiddlewareFunction) []medusahttp.MiddlewareFunction { return fns }

// Middlewares mirrors adminPaymentRoutesMiddlewares.
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/payments", Middlewares: mw(httputils.ValidateAndTransformQuery(AdminGetPaymentsParams, ListConfig))},
	{Methods: []string{"GET"}, Matcher: "/admin/payments/payment-providers", Middlewares: mw(httputils.ValidateAndTransformQuery(AdminGetPaymentProvidersParams, ProvidersListConfig))},
	{Methods: []string{"GET"}, Matcher: "/admin/payments/:id", Middlewares: mw(httputils.ValidateAndTransformQuery(AdminGetPaymentParams, RetrieveConfig))},
	{Methods: []string{"POST"}, Matcher: "/admin/payments/:id/capture", Middlewares: mw(httputils.ValidateAndTransformBody(httputils.Schema(AdminCreatePaymentCapture)), httputils.ValidateAndTransformQuery(AdminGetPaymentParams, RetrieveConfig))},
	{Methods: []string{"POST"}, Matcher: "/admin/payments/:id/refund", Middlewares: mw(httputils.ValidateAndTransformBody(httputils.Schema(AdminCreatePaymentRefund)), httputils.ValidateAndTransformQuery(AdminGetPaymentParams, RetrieveConfig))},
}

// RefetchPayment mirrors refetchPayment.
func RefetchPayment(req *medusahttp.MedusaRequest, id string) (any, error) {
	f := dal.NewRecord()
	f.Set("id", id)
	v := dal.NewRecord()
	v.Set("filters", f)
	r, err := httputils.RemoteQueryList(req.Ctx(), req.Scope, "payment", v, req.QueryConfig.Fields)
	if err != nil || len(r.Data) == 0 {
		return nil, err
	}
	return r.Data[0], nil
}

// Respond writes { payment }.
func Respond(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, id string) error {
	p, err := RefetchPayment(req, id)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	if p != nil {
		out.Set("payment", p)
	}
	return res.Status(200).JSON(out)
}

func list(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, entry, key string) error {
	filters := req.FilterableFields
	if filters == nil {
		filters = dal.NewRecord()
	}
	r, err := httputils.RemoteQueryList(req.Ctx(), req.Scope, entry, httputils.ListVariables(filters, req.QueryConfig.Pagination), req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	return res.JSON(httputils.ListResponse(key, r))
}

// GET mirrors GET /admin/payments.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	return list(req, res, "payment", "payments")
}

// ListProviders mirrors GET /admin/payments/payment-providers.
func ListProviders(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	return list(req, res, "payment_provider", "payment_providers")
}

// Route is route.ts.
var Route = medusahttp.Route{GET: GET}
