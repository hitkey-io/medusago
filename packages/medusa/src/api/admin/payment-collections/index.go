// Package paymentcollections mirrors api/admin/payment-collections.
package paymentcollections

import (
	orderworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
)

// DefaultPaymentCollectionFields mirrors defaultPaymentCollectionFields.
var DefaultPaymentCollectionFields = []string{"id", "currency_code", "amount", "status", "authorized_amount", "captured_amount", "refunded_amount", "*payment_sessions", "*payments"}

// RetrieveConfig mirrors retrievePaymentCollectionTransformQueryConfig.
var RetrieveConfig = httputils.QueryConfig{Defaults: DefaultPaymentCollectionFields, Entity: "payment_collection"}

// AdminGetPaymentCollectionParams mirrors AdminGetPaymentCollectionParams.
var AdminGetPaymentCollectionParams = utils.CreateSelectParams()

// AdminCreatePaymentCollection mirrors AdminCreatePaymentCollection.
var AdminCreatePaymentCollection = zod.Object(zod.F("order_id", zod.String()), zod.F("amount", zod.Number())).Strict()

// AdminMarkPaymentCollectionPaid mirrors AdminMarkPaymentCollectionPaid.
var AdminMarkPaymentCollectionPaid = zod.Object(zod.F("order_id", zod.String()), zod.F("provider_id", zod.String().Optional())).Strict()

// AdminInitializePaymentSession mirrors AdminInitializePaymentSession.
var AdminInitializePaymentSession = zod.Object(zod.F("provider_id", zod.String()), zod.F("data", zod.Record(zod.String(), zod.Unknown()).Optional())).Strict()

func mw(fns ...medusahttp.MiddlewareFunction) []medusahttp.MiddlewareFunction { return fns }

func q() medusahttp.MiddlewareFunction {
	return httputils.ValidateAndTransformQuery(AdminGetPaymentCollectionParams, RetrieveConfig)
}

// Middlewares mirrors adminPaymentCollectionsMiddlewares.
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"POST"}, Matcher: "/admin/payment-collections", Middlewares: mw(httputils.ValidateAndTransformBody(httputils.Schema(AdminCreatePaymentCollection)), q())},
	{Methods: []string{"POST"}, Matcher: "/admin/payment-collections/:id/mark-as-paid", Middlewares: mw(httputils.ValidateAndTransformBody(httputils.Schema(AdminMarkPaymentCollectionPaid)), q())},
	{Methods: []string{"POST"}, Matcher: "/admin/payment-collections/:id/payment-sessions", Middlewares: mw(httputils.ValidateAndTransformBody(httputils.Schema(AdminInitializePaymentSession)), q())},
}

// Respond writes { payment_collection } refetched with refetchEntity.
func Respond(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, id any) error {
	pc, err := httputils.RefetchEntity(req.Ctx(), req.Scope, "payment_collection", id, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("payment_collection", pc)
	return res.Status(200).JSON(out)
}

// POST mirrors POST /admin/payment-collections.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	created, err := orderworkflows.CreateOrderPaymentCollectionWorkflow.Run(req.Ctx(), req.Scope, orders.BodyCopy(req))
	if err != nil {
		return err
	}
	id, _ := created[0].Get("id")
	return Respond(req, res, id)
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
