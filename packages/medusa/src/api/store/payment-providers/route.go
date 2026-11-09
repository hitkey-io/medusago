// Package paymentproviders mirrors api/store/payment-providers.
package paymentproviders

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	storeutils "github.com/hitkey-io/medusago/packages/medusa/src/api/store/utils"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
)

// DefaultFields mirrors defaultAdminPaymentProviderFields.
var DefaultFields = []string{"id", "is_enabled"}

// ListConfig mirrors listTransformPaymentProvidersQueryConfig.
var ListConfig = httputils.QueryConfig{Defaults: DefaultFields, Allowed: storeutils.BuildAllowedFields(DefaultFields), IsList: true}

// StoreGetPaymentProvidersParams mirrors StoreGetPaymentProvidersParams.
var StoreGetPaymentProvidersParams = utils.CreateFindParams(utils.FindParamsOptions{Offset: utils.Num(0), Limit: utils.Num(20)}).
	Merge(zod.Object(zod.F("region_id", zod.String())))

// Middlewares mirrors storePaymentProvidersMiddlewares.
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/store/payment-providers", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(StoreGetPaymentProvidersParams, ListConfig)}},
}

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	regionID, _ := req.FilterableFields.Get("region_id")
	if regionID == nil || regionID == "" {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, "You must provide the region_id to list payment providers")
	}
	f := dal.NewRecord()
	f.Set("region_id", regionID)
	fields := make([]string, len(req.QueryConfig.Fields))
	for i, x := range req.QueryConfig.Fields {
		fields[i] = "payment_provider." + x
	}
	r, err := httputils.RemoteQueryList(req.Ctx(), req.Scope, "region_payment_provider", httputils.ListVariables(f, req.QueryConfig.Pagination), fields)
	if err != nil {
		return err
	}
	providers := make([]any, 0, len(r.Data))
	for _, row := range r.Data {
		var p any
		if rec, ok := row.(*dal.Record); ok {
			p, _ = rec.Get("payment_provider")
		}
		providers = append(providers, p)
	}
	r.Data = providers
	return res.JSON(httputils.ListResponse("payment_providers", r))
}

// Route is route.ts.
var Route = medusahttp.Route{GET: GET}
