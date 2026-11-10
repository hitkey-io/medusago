// Package paymentcollections mirrors api/store/payment-collections.
package paymentcollections

import (
	cartworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	storeutils "github.com/hitkey-io/medusago/packages/medusa/src/api/store/utils"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
)

// DefaultPaymentCollectionFields mirrors defaultPaymentCollectionFields.
var DefaultPaymentCollectionFields = []string{"id", "currency_code", "amount", "*payment_sessions"}

// RetrieveConfig mirrors retrievePaymentCollectionTransformQueryConfig.
var RetrieveConfig = httputils.QueryConfig{Defaults: DefaultPaymentCollectionFields, Allowed: storeutils.BuildAllowedFields(DefaultPaymentCollectionFields, []string{"status", "payment_providers", "payment_providers.id", "payment_sessions.id", "payment_sessions.amount", "payment_sessions.currency_code", "payment_sessions.provider_id", "payment_sessions.data", "payment_sessions.status"})}

// Validators.
var (
	StoreCreatePaymentCollection = zod.Object(zod.F("cart_id", zod.String())).Strict()
	StoreCreatePaymentSession    = zod.Object(
		zod.F("provider_id", zod.String()),
		zod.F("data", zod.Record(zod.String(), zod.Unknown()).Optional()),
	).Strict()
)

func mw(fns ...medusahttp.MiddlewareFunction) []medusahttp.MiddlewareFunction { return fns }

// Middlewares mirrors storePaymentCollectionsMiddlewares.
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"POST"}, Matcher: "/store/payment-collections", Middlewares: mw(httputils.ValidateAndTransformBody(httputils.Schema(StoreCreatePaymentCollection)), httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig))},
	{Methods: []string{"POST"}, Matcher: "/store/payment-collections/:id/payment-sessions", Middlewares: mw(httputils.ValidateAndTransformBody(httputils.Schema(StoreCreatePaymentSession)), httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig))},
}

func collectionForCart(req *medusahttp.MedusaRequest, cartID any) (any, error) {
	f := dal.NewRecord()
	f.Set("cart_id", cartID)
	v := dal.NewRecord()
	v.Set("filters", f)
	fields := make([]string, len(req.QueryConfig.Fields))
	for i, x := range req.QueryConfig.Fields {
		fields[i] = "payment_collection." + x
	}
	r, err := httputils.RemoteQueryList(req.Ctx(), req.Scope, "cart_payment_collection", v, fields)
	if err != nil {
		return nil, err
	}
	if len(r.Data) == 0 {
		return nil, nil
	}
	rel, _ := r.Data[0].(*dal.Record)
	if rel == nil {
		return nil, nil
	}
	pc, _ := rel.Get("payment_collection")
	return pc, nil
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	cartID, _ := body.Get("cart_id")
	pc, err := collectionForCart(req, cartID)
	if err != nil {
		return err
	}
	if pc == nil {
		id, _ := cartID.(string)
		if _, err := cartworkflows.CreatePaymentCollectionForCartWorkflow.Run(req.Ctx(), req.Scope, id); err != nil {
			return err
		}
		if pc, err = collectionForCart(req, cartID); err != nil {
			return err
		}
	}
	out := dal.NewRecord()
	out.Set("payment_collection", pc)
	return res.Status(200).JSON(out)
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
